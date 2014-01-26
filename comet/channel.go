package main

import (
	"container/list"
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"net"
	"sync"
)

var (
	ErrChannelNotExist = errors.New("Channle not exist")
)

var (
	UserChannel *ChannelList
)

// The subscriber interface.
type Channel interface {
	// PushMsg push a message to the subscriber.
	PushMsg(key string, m *Message) error
	// Add a token for one subscriber
	// The request token not equal the subscriber token will return errors.
	AddToken(key, token string) error
	// Auth auth the access token.
	// The request token not match the subscriber token will return errors.
	AuthToken(key, token string) bool
	// AddConn add a connection for the subscriber.
	// Exceed the max number of subscribers per key will return errors.
	AddConn(key string, conn net.Conn) (*list.Element, error)
	// RemoveConn remove a connection for the  subscriber.
	RemoveConn(key string, e *list.Element) error
	// Expire expire the channle and clean data.
	Close() error
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
	Log.Debug("create %d ChannelBucket", Conf.ChannelBucket)
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
	Log.Debug("user_key:\"%s\" hit channel bucket index:%d", key, idx)
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
		Log.Info("user_key:\"%s\" refresh channel bucket expire time", key)
		return c, nil
	} else {
		c = NewSeqChannel()
		b.Data[key] = c
		b.Unlock()
		ChStat.IncrCreate()
		Log.Info("user_key:\"%s\" create a new channel", key)
		return c, nil
	}
}

// Get a user channel from ChannleList.
func (l *ChannelList) Get(key string) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	if c, ok := b.Data[key]; !ok {
		if !Conf.Auth {
			c = NewSeqChannel()
			b.Data[key] = c
			b.Unlock()
			ChStat.IncrCreate()
			Log.Info("user_key:\"%s\" create a new channel", key)
			return c, nil
		} else {
			b.Unlock()
			Log.Warn("user_key:\"%s\" channle not exists", key)
			return nil, ErrChannelNotExist
		}
	} else {
		b.Unlock()
		ChStat.IncrAccess()
		Log.Info("user_key:\"%s\" refresh channel bucket expire time", key)
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
		Log.Warn("user_key:\"%s\" channle not exists", key)
		return nil, ErrChannelNotExist
	} else {
		delete(b.Data, key)
		b.Unlock()
		ChStat.IncrDelete()
		Log.Info("user_key:\"%s\" delete channel", key)
		return c, nil
	}
}

// Close close all channel.
func (l *ChannelList) Close() {
	Log.Info("channel close")
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
			Log.Error("c.Close() error(%v)", err)
		}
	}
}
