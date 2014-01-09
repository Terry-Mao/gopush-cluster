package main

import (
	"container/list"
	"errors"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"sync"
	"time"
)

var (
	ErrMessageSave = errors.New("Message set failed")
	ErrMessageGet  = errors.New("Message get failed")
)

type OuterChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn *list.List
	// Channel expired unixnano
	expire int64
	// token
	token *Token
	// time id
	timeID *TimeID
}

// New a user outer stored message channel.
func NewOuterChannel() *OuterChannel {
	var t *Token

	if Conf.Auth == 1 {
		t = NewToken()
	} else {
		t = nil
	}

	return &OuterChannel{
		mutex:  &sync.Mutex{},
		conn:   list.New(),
		expire: time.Now().UnixNano() + Conf.ChannelExpireSec*Second,
		timeID: NewTimeID(),
		token:  t,
	}
}

// AddToken implements the Channel AddToken method.
func (c *OuterChannel) AddToken(token string, expire int64, key string) error {
	c.mutex.Lock()
	if err := c.token.Add(token, expire); err != nil {
		Log.Error("user_key:\"%s\" add token failed (%s)", key, err.Error())
		c.mutex.Unlock()
		return err
	}

	c.mutex.Unlock()
	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *OuterChannel) AuthToken(token string, key string) error {
	c.mutex.Lock()
	if err := c.token.Auth(token); err != nil {
		Log.Error("user_key:\"%s\" auth token failed (%s)", key, err.Error())
		c.mutex.Unlock()
		return err
	}

	c.mutex.Unlock()
	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *OuterChannel) PushMsg(m *Message, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// rewrite message id
	m.MsgID = c.timeID.ID()
	Log.Debug("user_key:\"%s\" timeID:%d", key, m.MsgID)
	args := &myrpc.MessageSaveArgs{MsgID: m.MsgID, Msg: m.Msg, Expire: m.Expire, Key: key}
	reply := retOK
	if err := MsgRPC.Call("MessageRPC.Save", args, &reply); err != nil {
		Log.Error("MessageRPC.Save failed (%s)", err.Error())
		return err
	}

	if reply != retOK {
		Log.Error("MessageRPC.Save failed (ret=%d)", reply)
		return ErrMessageSave
	}

	// send message to each conn when message id > conn last message id
	b, err := m.Bytes()
	if err != nil {
		Log.Error("message.Bytes() failed (%s)", err.Error())
		return err
	}
	// push message
	for e := c.conn.Front(); e != nil; e = e.Next() {
		conn, _ := e.Value.(net.Conn)
		// do something with e.Value
		if n, err := conn.Write(b); err != nil {
			Log.Error("conn.Write() failed (%s)", err.Error())
			continue
		} else {
			Log.Debug("PushMsg conn.Write %d bytes (%v)", n, b)
		}

		Log.Info("user_key:\"%s\" push message \"%s\":%d", key, m.Msg, m.MsgID)

	}

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *OuterChannel) AddConn(conn net.Conn, key string) (*list.Element, error) {
	c.mutex.Lock()
	if c.conn.Len()+1 > Conf.MaxSubscriberPerKey {
		Log.Error("user_key:\"%s\" exceed conn", key)
		return nil, ErrMaxConn
	}

	e := c.conn.PushBack(conn)
	c.mutex.Unlock()
	Log.Debug("user_key:\"%s\" add conn", key)
	ConnStat.IncrAdd()
	return e, nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *OuterChannel) RemoveConn(e *list.Element, key string) error {
	c.mutex.Lock()
	c.conn.Remove(e)
	c.mutex.Unlock()
	Log.Debug("user_key:\"%s\" remove conn", key)
	ConnStat.IncrRemove()
	return nil
}

// SetDeadline implements the Channel SetDeadline method.
func (c *OuterChannel) SetDeadline(d int64) {
	c.expire = d
}

// Timeout implements the Channel Timeout method.
func (c *OuterChannel) Timeout() bool {
	return time.Now().UnixNano() > c.expire
}

// Close implements the Channel Close method.
func (c *OuterChannel) Close() error {
	mutex := c.mutex
	mutex.Lock()
	for e := c.conn.Front(); e != nil; e = e.Next() {
		conn, _ := e.Value.(net.Conn)
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Warn("conn.Close() failed (%s)", err.Error())
		}
	}

	c.conn = nil
	c.token = nil
	c.timeID = nil
	c.mutex = nil
	mutex.Unlock()
	return nil
}
