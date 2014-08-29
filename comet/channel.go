// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	log "code.google.com/p/log4go"
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"github.com/Terry-Mao/gopush-cluster/hlist"
	"github.com/Terry-Mao/gopush-cluster/ketama"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"sync"
)

var (
	ErrChannelNotExist = errors.New("Channle not exist")
	ErrConnProto       = errors.New("Unknown connection protocol")
	UserChannel        *ChannelList
	CometRing          *ketama.HashRing
)

// The subscriber interface.
type Channel interface {
	// PushMsg push a message to the subscriber.
	PushMsg(key string, m *myrpc.Message, expire uint) error
	// Add a token for one subscriber
	// The request token not equal the subscriber token will return errors.
	AddToken(key, token string) error
	// Auth auth the access token.
	// The request token not match the subscriber token will return errors.
	AuthToken(key, token string) bool
	// AddConn add a connection for the subscriber.
	// Exceed the max number of subscribers per key will return errors.
	AddConn(key string, conn *Connection) (*hlist.Element, error)
	// RemoveConn remove a connection for the  subscriber.
	RemoveConn(key string, e *hlist.Element) error
	// Expire expire the channle and clean data.
	Close() error
}

// Connection
type Connection struct {
	Conn    net.Conn
	Proto   uint8
	Version string
	Buf     chan []byte
}

// HandleWrite start a goroutine get msg from chan, then send to the conn.
func (c *Connection) HandleWrite(key string) {
	go func() {
		var (
			n   int
			err error
		)
		log.Debug("user_key: \"%s\" HandleWrite goroutine start", key)
		for {
			msg, ok := <-c.Buf
			if !ok {
				log.Debug("user_key: \"%s\" HandleWrite goroutine stop", key)
				return
			}
			if c.Proto == WebsocketProto {
				// raw
				n, err = c.Conn.Write(msg)
			} else if c.Proto == TCPProto {
				// redis protocol
				msg = []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), string(msg)))
				n, err = c.Conn.Write(msg)
			} else {
				log.Error("unknown connection protocol: %d", c.Proto)
				panic(ErrConnProto)
			}
			// update stat
			if err != nil {
				log.Error("user_key: \"%s\" conn.Write() error(%v)", key, err)
				MsgStat.IncrFailed(1)
			} else {
				log.Debug("user_key: \"%s\" write \r\n========%s(%d)========", key, string(msg), n)
				MsgStat.IncrSucceed(1)
			}
		}
	}()
}

// Write different message to client by different protocol
func (c *Connection) Write(key string, msg []byte) {
	select {
	case c.Buf <- msg:
	default:
		c.Conn.Close()
		log.Warn("user_key: \"%s\" discard message: \"%s\" and close connection", key, string(msg))
	}
}

// Channel bucket.
type ChannelBucket struct {
	Data  map[string]Channel
	mutex *sync.Mutex
}

// Channel list.
type ChannelList struct {
	Channels []*ChannelBucket
}

// Lock lock the bucket mutex.
func (c *ChannelBucket) Lock() {
	c.mutex.Lock()
}

// Unlock unlock the bucket mutex.
func (c *ChannelBucket) Unlock() {
	c.mutex.Unlock()
}

// NewChannelList create a new channel bucket set.
func NewChannelList() *ChannelList {
	l := &ChannelList{Channels: []*ChannelBucket{}}
	// split hashmap to many bucket
	log.Debug("create %d ChannelBucket", Conf.ChannelBucket)
	for i := 0; i < Conf.ChannelBucket; i++ {
		c := &ChannelBucket{
			Data:  map[string]Channel{},
			mutex: &sync.Mutex{},
		}
		l.Channels = append(l.Channels, c)
	}
	return l
}

// Count get the bucket total channel count.
func (l *ChannelList) Count() int {
	c := 0
	for i := 0; i < Conf.ChannelBucket; i++ {
		c += len(l.Channels[i].Data)
	}
	return c
}

// bucket return a channelBucket use murmurhash3.
func (l *ChannelList) bucket(key string) *ChannelBucket {
	h := hash.NewMurmur3C()
	h.Write([]byte(key))
	idx := uint(h.Sum32()) & uint(Conf.ChannelBucket-1)
	log.Debug("user_key:\"%s\" hit channel bucket index:%d", key, idx)
	return l.Channels[idx]
}

// New create a user channle.
func (l *ChannelList) New(key string) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	if c, ok := b.Data[key]; ok {
		b.Unlock()
		ChStat.IncrAccess()
		log.Info("user_key:\"%s\" refresh channel bucket expire time", key)
		return c, nil
	} else {
		c = NewSeqChannel()
		b.Data[key] = c
		b.Unlock()
		ChStat.IncrCreate()
		log.Info("user_key:\"%s\" create a new channel", key)
		return c, nil
	}
}

// Get a user channel from ChannleList.
func (l *ChannelList) Get(key string, newOne bool) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	if c, ok := b.Data[key]; !ok {
		if !Conf.Auth && newOne {
			c = NewSeqChannel()
			b.Data[key] = c
			b.Unlock()
			ChStat.IncrCreate()
			log.Info("user_key:\"%s\" create a new channel", key)
			return c, nil
		} else {
			b.Unlock()
			log.Warn("user_key:\"%s\" channle not exists", key)
			return nil, ErrChannelNotExist
		}
	} else {
		b.Unlock()
		ChStat.IncrAccess()
		log.Info("user_key:\"%s\" refresh channel bucket expire time", key)
		return c, nil
	}
}

// Delete a user channel from ChannleList.
func (l *ChannelList) Delete(key string) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	if c, ok := b.Data[key]; !ok {
		b.Unlock()
		log.Warn("user_key:\"%s\" delete channle not exists", key)
		return nil, ErrChannelNotExist
	} else {
		delete(b.Data, key)
		b.Unlock()
		ChStat.IncrDelete()
		log.Info("user_key:\"%s\" delete channel", key)
		return c, nil
	}
}

// Close close all channel.
func (l *ChannelList) Close() {
	log.Info("channel close")
	chs := make([]Channel, 0, l.Count())
	for _, c := range l.Channels {
		c.Lock()
		for _, c := range c.Data {
			chs = append(chs, c)
		}
		c.Unlock()
	}
	// close all channels
	for _, c := range chs {
		if err := c.Close(); err != nil {
			log.Error("c.Close() error(%v)", err)
		}
	}
}

// Migrate migrate portion of connections which don`t belong to this Comet
func (l *ChannelList) Migrate() {
	// init ketama
	ring := ketama.NewRing(Conf.KetamaBase)
	for node, info := range cometNodeInfoMap {
		ring.AddNode(node, info.Weight)
	}
	ring.Bake()
	CometRing = ring

	// get all the channel lock
	channels := []Channel{}
	for i, c := range l.Channels {
		c.Lock()
		for k, v := range c.Data {
			hn := ring.Hash(k)
			if hn != Conf.ZookeeperCometNode {
				channels = append(channels, v)
				delete(c.Data, k)
				log.Debug("migrate delete channel key \"%s\"", k)
			}
		}
		c.Unlock()
		log.Debug("migrate channel bucket:%d finished", i)
	}
	// close all the migrate channels
	log.Info("close all the migrate channels")
	for _, channel := range channels {
		if err := channel.Close(); err != nil {
			log.Error("channel.Close() error(%v)", err)
			continue
		}
	}
	log.Info("close all the migrate channels finished")
}
