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
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hlist"
	"github.com/Terry-Mao/gopush-cluster/id"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/golang/glog"
	"sync"
)

var (
	ErrMessageSave   = errors.New("Message set failed")
	ErrMessageGet    = errors.New("Message get failed")
	ErrMessageRPC    = errors.New("Message RPC not init")
	ErrAssectionConn = errors.New("Assection type Connection failed")
)

// Sequence Channel struct.
type SeqChannel struct {
	// Mutex
	mutex *sync.Mutex
	// client conn double linked-list
	conn *hlist.Hlist
	// TODO Remove time id or lazy New
	timeID *id.TimeID
	// token
	token *Token
}

// New a user seq stored message channel.
func NewSeqChannel() *SeqChannel {
	ch := &SeqChannel{
		mutex:  &sync.Mutex{},
		conn:   hlist.New(),
		timeID: id.NewTimeID(),
		token:  nil,
	}
	// save memory
	if Conf.Auth {
		ch.token = NewToken()
	}
	return ch
}

// AddToken implements the Channel AddToken method.
func (c *SeqChannel) AddToken(key, token string) error {
	if !Conf.Auth {
		return nil
	}
	c.mutex.Lock()
	if err := c.token.Add(token); err != nil {
		c.mutex.Unlock()
		glog.Errorf("user_key:\"%s\" c.token.Add(\"%s\") error(%v)", key, token, err)
		return err
	}
	c.mutex.Unlock()
	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *SeqChannel) AuthToken(key, token string) bool {
	if !Conf.Auth {
		return true
	}
	c.mutex.Lock()
	if err := c.token.Auth(token); err != nil {
		c.mutex.Unlock()
		glog.Errorf("user_key:\"%s\" c.token.Auth(\"%s\") error(%v)", key, token, err)
		return false
	}
	c.mutex.Unlock()
	return true
}

// PushMsg implements the Channel PushMsg method.
func (c *SeqChannel) PushMsg(key string, m *Message) error {
	var succeed, failed uint64
	client := MessageRPC.Get()
	if client == nil {
		return ErrMessageRPC
	}
	succeed, failed = 0, 0
	c.mutex.Lock()
	// private message need persistence
	// if message expired no need persistence, only send online message
	if m.GroupID != myrpc.PublicGroupID && m.Expire > 0 {
		// rewrite message id
		m.MsgID = c.timeID.ID()
		glog.V(1).Infof("user_key:\"%s\" timeID:%d", key, m.MsgID)
		args := &myrpc.MessageSaveArgs{MsgID: m.MsgID, Msg: m.Msg, Expire: m.Expire, Key: key}
		reply := myrpc.OK
		if err := client.Call("MessageRPC.Save", args, &reply); err != nil {
			c.mutex.Unlock()
			glog.Errorf("MessageRPC.Save(\"%s\", %v) error(%v)", key, m, err)
			return err
		}
		// message save failed
		if reply != myrpc.OK {
			c.mutex.Unlock()
			glog.Errorf("MessageRPC.Save(\"%s\", %v) error(ret=%d)", key, m, reply)
			return ErrMessageSave
		}
	}
	// send message to each conn when message id > conn last message id
	b, err := m.Bytes()
	if err != nil {
		c.mutex.Unlock()
		return err
	}
	// push message
	for e := c.conn.Front(); e != nil; e = e.Next() {
		conn, _ := e.Value.(*Connection)
		// do something with e.Value
		if n, err := conn.Write(b); err != nil {
			glog.Errorf("user_key:\"%s\" conn.Write() error(%v)", key, err)
			failed++
			continue
		} else {
			succeed++
			glog.Errorf("user_key:\"%s\" conn.Write %d bytes", key, n)
		}
	}
	c.mutex.Unlock()
	glog.Infof("user_key:\"%s\" push message \"%s\":%d, (succeed:%d, failed:%d)", key, m.Msg, m.MsgID, succeed, failed)
	// message stat
	MsgStat.IncrFailed(failed)
	MsgStat.IncrSucceed(succeed)
	return nil
}

// AddConn implements the Channel AddConn method.
func (c *SeqChannel) AddConn(key string, conn *Connection) (*hlist.Element, error) {
	c.mutex.Lock()
	if c.conn.Len()+1 > Conf.MaxSubscriberPerChannel {
		c.mutex.Unlock()
		glog.Errorf("user_key:\"%s\" exceed conn", key)
		return nil, ErrMaxConn
	}
	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err := conn.Conn.Write(HeartbeatReply); err != nil {
		c.mutex.Unlock()
		glog.Errorf("user_key:\"%s\" write first heartbeat to client error(%v)", key, err)
		return nil, err
	}
	// add conn
	e := c.conn.PushFront(conn)
	c.mutex.Unlock()
	ConnStat.IncrAdd()
	glog.Infof("user_key:\"%s\" add conn = %d", key, c.conn.Len())
	return e, nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *SeqChannel) RemoveConn(key string, e *hlist.Element) error {
	c.mutex.Lock()
	c.conn.Remove(e)
	c.mutex.Unlock()
	ConnStat.IncrRemove()
	glog.Infof("user_key:\"%s\" remove conn = %d", key, c.conn.Len())
	return nil
}

// Close implements the Channel Close method.
func (c *SeqChannel) Close() error {
	c.mutex.Lock()
	for e := c.conn.Front(); e != nil; e = e.Next() {
		if conn, ok := e.Value.(*Connection); !ok {
			c.mutex.Unlock()
			return ErrAssectionConn
		} else {
			if err := conn.Conn.Close(); err != nil {
				// ignore close error
				glog.Warningf("conn.Close() error(%v)", err)
			}
		}
	}
	c.mutex.Unlock()
	return nil
}
