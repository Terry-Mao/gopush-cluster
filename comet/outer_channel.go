package main

import (
	"errors"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"sync"
	"time"
)

var (
	MessageSaveErr = errors.New("Message set failed")
)

type OuterChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]int64
	// Channel expired unixnano
	expire int64
	// token
	token *Token
	// snowflake id
	snowflake *Snowflake
}

// New a user outer stored message channel.
func NewOuterChannel() *OuterChannel {
	return &OuterChannel{
		mutex:     &sync.Mutex{},
		conn:      map[net.Conn]int64{},
		expire:    time.Now().UnixNano() + Conf.ChannelExpireSec*Second,
		snowflake: NewSnowflake(),
		token:     NewToken(),
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

// SendOfflineMsg implements the Channel SendOfflineMsg method.
func (c *OuterChannel) SendOfflineMsg(conn net.Conn, mid int64, key string) error {
	// no need
	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *OuterChannel) PushMsg(m *Message, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// rewrite message id
	m.MsgID = c.snowflake.ID()
	Log.Debug("user_key:\"%s\" snowflakeID:%d", key, m.MsgID)
	args := &myrpc.MessageSaveArgs{MsgID: m.MsgID, Msg: m.Msg, Expire: m.Expire, Key: key}
	reply := retOK
	if err := RPCCli.Call("MessageRPC.Save", args, &reply); err != nil {
		Log.Error("MessageRPC.Save failed (%s)", err.Error())
		return err
	}

	if reply != retOK {
		Log.Error("MessageRPC.Save failed (ret=%d)", reply)
		return MessageSaveErr
	}

	// send message to each conn when message id > conn last message id
	b, err := m.Bytes()
	if err != nil {
		Log.Error("message.Bytes() failed (%s)", err.Error())
		return err
	}
	// push message
	for conn, mid := range c.conn {
		// ignore message cause it's id less than mid
		if mid >= m.MsgID {
			Log.Warn("user_key:\"%s\" ignore mid:%d, last mid:%d", key, m.MsgID, mid)
			continue
		}

		if n, err := conn.Write(b); err != nil {
			Log.Error("conn.Write() failed (%s)", err.Error())
			continue
		} else {
			Log.Debug("PushMsg conn.Write %d bytes (%v)", n, b)
		}

		// if succeed, update the last message id, conn.Write may failed but err == nil(client shutdown or sth else), but the message won't loss till next connect to sub
		c.conn[conn] = m.MsgID
		Log.Info("user_key:\"%s\" push message \"%s\":%d", key, m.Msg, m.MsgID)
	}

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *OuterChannel) AddConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	// save the last push message id
	c.conn[conn] = mid
	c.mutex.Unlock()
	Log.Debug("user_key:\"%s\" add conn", key)
	ConnStat.IncrAdd()
	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *OuterChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	delete(c.conn, conn)
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

	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Warn("conn.Close() failed (%s)", err.Error())
		}
	}

	c.conn = nil
	c.token = nil
	c.snowflake = nil
	c.mutex = nil
	c = nil
	mutex.Unlock()
	return nil
}
