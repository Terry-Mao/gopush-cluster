package main

import (
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"net"
	"sync"
	"time"
)

const (
	Second           = int64(time.Second)
	InnerChannelType = 1
	OuterChannelType = 2
)

var (
	// Channle not exists
	ChannelNotExistErr = errors.New("Channle not exist")
	// Channel expired
	ChannelExpiredErr = errors.New("Channel expired")
	// Channle type unknown
	ChannelTypeErr = errors.New("Channle type unknown")
)

// The subscriber interface
type Channel interface {
	// SendOfflineMsg send messages which id greate than the request id to the subscriber.
	// net.Conn write failed will return errors.
	SendOfflineMsg(conn net.Conn, mid int64, key string) error
	// PushMsg push a message to the subscriber.
	PushMsg(m *Message, key string) error
	// AddConn add a connection for the subscriber.
	// Exceed the max number of subscribers per key will return errors.
	AddConn(conn net.Conn, mid int64, key string) error
	// RemoveConn remove a connection for the  subscriber.
	RemoveConn(conn net.Conn, mid int64, key string) error
	// SetDeadline set the channel deadline unixnano
	SetDeadline(d int64)
	// Timeout
	Timeout() bool
	// Expire expire the channle and clean data.
	Close() error
}

type ChannelBucket struct {
	Data  map[string]Channel
	mutex *sync.Mutex
}

type ChannelList struct {
	Channels []*ChannelBucket
}

var (
	UserChannel *ChannelList
)

func (c *ChannelBucket) Lock() {
	c.mutex.Lock()
}

func (c *ChannelBucket) Unlock() {
	c.mutex.Unlock()
}

func NewChannelList() *ChannelList {
	l := &ChannelList{Channels: []*ChannelBucket{}}
	// split hashmap to many bucket
	for i := 0; i < Conf.ChannelBucket; i++ {
		c := &ChannelBucket{
			Data:  map[string]Channel{},
			mutex: &sync.Mutex{},
		}

		l.Channels = append(l.Channels, c)
	}

	return l
}

// bucket return a channelBucket use murmurhash3
func (l *ChannelList) bucket(key string) *ChannelBucket {
	h := hash.NewMurmur3C()
	h.Write([]byte(key))
	idx := uint(h.Sum32()) & uint(Conf.ChannelBucket-1)
	return l.Channels[idx]
}

// New create a user channle
func (l *ChannelList) New(key string) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	defer b.Unlock()

	if c, ok := b.Data[key]; ok {
		// refresh the expire time
		c.SetDeadline(time.Now().UnixNano() + Conf.ChannelExpireSec*Second)
		return c, nil
	} else {
		if Conf.ChannelType == InnerChannelType {
			c = NewInnerChannel()
		} else if Conf.ChannelType == OuterChannelType {
			c = NewOuterChannel()
		} else {
			Log.Error("user_key:\"%s\" unknown channel type : %d (0: inner_channel, 1: outer_channel)", key, Conf.ChannelType)
			return nil, ChannelTypeErr
		}

		b.Data[key] = c
		return c, nil
	}
}

// Get a user channel from ChannleList
func (l *ChannelList) Get(key string) (Channel, error) {
	// get a channel bucket
	b := l.bucket(key)
	b.Lock()
	defer b.Unlock()

	if c, ok := b.Data[key]; !ok {
		Log.Warn("user_key:\"%s\" channle not exists", key)
		return nil, ChannelNotExistErr
	} else {
		// check expired
		if c.Timeout() {
			Log.Warn("user_key:\"%s\" channle expired", key)
			delete(b.Data, key)
			if err := c.Close(); err != nil {
				Log.Error("user_key:\"%s\" channel close failed (%s)", key, err.Error())
				return nil, err
			}

			return nil, ChannelExpiredErr
		}

		return c, nil
	}
}
