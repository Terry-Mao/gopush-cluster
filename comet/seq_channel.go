package main

import (
	"container/list"
	"errors"
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
	conn *list.List
	// TODO Remove time id
	timeID *TimeID
	// token
	token *Token
}

// New a user seq stored message channel.
func NewSeqChannel() *SeqChannel {
	ch := &SeqChannel{
		mutex:  &sync.Mutex{},
		conn:   list.New(),
		timeID: NewTimeID(),
		token:  nil,
	}
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
		Log.Error("user_key:\"%s\" c.token.Add(\"%s\") failed (%s)", key, token, err.Error())
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
		Log.Error("user_key:\"%s\" c.token.Auth(\"%s\") failed (%s)", key, token, err.Error())
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
			Log.Error("MessageRPC.Save failed (%s)", err.Error())
			return err
		}
		// message save failed
		if reply != retOK {
			c.mutex.Unlock()
			Log.Error("MessageRPC.Save failed (ret=%d)", reply)
			return ErrMessageSave
		}
	}
	// send message to each conn when message id > conn last message id
	b, err := m.Bytes()
	if err != nil {
		c.mutex.Unlock()
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
			Log.Debug("conn.Write %d bytes (%v)", n, b)
		}
		Log.Info("user_key:\"%s\" push message \"%s\":%d", key, m.Msg, m.MsgID)
	}
	c.mutex.Unlock()
	return nil
}

// AddConn implements the Channel AddConn method.
func (c *SeqChannel) AddConn(key string, conn net.Conn) (*list.Element, error) {
	c.mutex.Lock()
	if c.conn.Len()+1 > Conf.MaxSubscriberPerChannel {
		c.mutex.Unlock()
		Log.Error("user_key:\"%s\" exceed conn", key)
		return nil, ErrMaxConn
	}
	e := c.conn.PushBack(conn)
	c.mutex.Unlock()
	ConnStat.IncrAdd()
	Log.Debug("user_key:\"%s\" add conn", key)
	return e, nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *SeqChannel) RemoveConn(key string, e *list.Element) error {
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
			Log.Warn("conn.Close() failed (%s)", err.Error())
		}
	}
	c.mutex.Unlock()
	return nil
}
