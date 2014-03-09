package main

import (
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hlist"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"sync"
)

var (
	ErrMessageSave = errors.New("Message set failed")
	ErrMessageGet  = errors.New("Message get failed")
	ErrMessageRPC  = errors.New("Message RPC not init")
)

// Sequence Channel struct.
type SeqChannel struct {
	// Mutex
	mutex *sync.Mutex
	// client conn double linked-list
	conn *hlist.Hlist
	// TODO Remove time id or lazy New
	timeID *TimeID
	// token
	token *Token
}

// New a user seq stored message channel.
func NewSeqChannel() *SeqChannel {
	ch := &SeqChannel{
		mutex:  &sync.Mutex{},
		conn:   hlist.New(),
		timeID: NewTimeID(),
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
		Log.Error("user_key:\"%s\" c.token.Add(\"%s\") error(%v)", key, token, err)
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
		Log.Error("user_key:\"%s\" c.token.Auth(\"%s\") error(%v)", key, token, err)
		return false
	}
	c.mutex.Unlock()
	return true
}

// PushMsg implements the Channel PushMsg method.
func (c *SeqChannel) PushMsg(key string, m *Message) error {
	if MsgRPC == nil {
		return ErrMessageRPC
	}
	c.mutex.Lock()
	// private message need persistence
	if m.GroupID != myrpc.PublicGroupID {
		// rewrite message id
		m.MsgID = c.timeID.ID()
		Log.Debug("user_key:\"%s\" timeID:%d", key, m.MsgID)
		args := &myrpc.MessageSaveArgs{MsgID: m.MsgID, Msg: m.Msg, Expire: m.Expire, Key: key}
		reply := retOK
		if err := MsgRPC.Call("MessageRPC.Save", args, &reply); err != nil {
			c.mutex.Unlock()
			Log.Error("MessageRPC.Save error(%v)", err)
			return err
		}
		// message save failed
		if reply != retOK {
			c.mutex.Unlock()
			Log.Error("MessageRPC.Save error(ret=%d)", reply)
			return ErrMessageSave
		}
	}
	// send message to each conn when message id > conn last message id
	b, err := m.Bytes()
	if err != nil {
		c.mutex.Unlock()
		Log.Error("message.Bytes() error(%v)", err)
		return err
	}
	// push message
	for e := c.conn.Front(); e != nil; e = e.Next() {
		conn, _ := e.Value.(net.Conn)
		// do something with e.Value
		if n, err := conn.Write(b); err != nil {
			Log.Error("conn.Write() error(%v)", err)
			continue
		} else {
			Log.Debug("conn.Write %d bytes", n)
		}
		Log.Info("user_key:\"%s\" push message \"%s\":%d", key, m.Msg, m.MsgID)
	}
	c.mutex.Unlock()
	return nil
}

// AddConn implements the Channel AddConn method.
func (c *SeqChannel) AddConn(key string, conn net.Conn) (*hlist.Element, error) {
	c.mutex.Lock()
	if c.conn.Len()+1 > Conf.MaxSubscriberPerChannel {
		c.mutex.Unlock()
		Log.Error("user_key:\"%s\" exceed conn", key)
		return nil, ErrMaxConn
	}
	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err := conn.Write(HeartbeatReply); err != nil {
		c.mutex.Unlock()
		Log.Error("user_key:\"%s\" write first heartbeat to client error(%v)", key, err)
		return nil, err
	}
	// add conn
	e := c.conn.PushFront(conn)
	c.mutex.Unlock()
	ConnStat.IncrAdd()
	Log.Debug("user_key:\"%s\" add conn", key)
	return e, nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *SeqChannel) RemoveConn(key string, e *hlist.Element) error {
	c.mutex.Lock()
	c.conn.Remove(e)
	c.mutex.Unlock()
	ConnStat.IncrRemove()
	Log.Debug("user_key:\"%s\" remove conn", key)
	return nil
}

// Close implements the Channel Close method.
func (c *SeqChannel) Close() error {
	c.mutex.Lock()
	for e := c.conn.Front(); e != nil; e = e.Next() {
		conn, _ := e.Value.(net.Conn)
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Warn("conn.Close() error(%v)", err)
		}
	}
	c.mutex.Unlock()
	return nil
}
