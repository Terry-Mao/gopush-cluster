package main

import (
	"github.com/Terry-Mao/gopush2/skiplist"
	"net"
	"sync"
	"time"
)

type InnerChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]bool
	// Stored message
	message *skiplist.SkipList
	// Subscriber expired unixnano
	expire int64
	// Max message stored number
	maxMessage int
}

// New a inner message stored channel
func NewInnerChannel() *InnerChannel {
	return &InnerChannel{
		mutex:      &sync.Mutex{},
		message:    skiplist.New(),
		conn:       map[net.Conn]bool{},
		maxMessage: Conf.MaxStoredMessage,
		expire:     time.Now().UnixNano() + Conf.ChannelExpireSec*Second,
	}
}

// SendOfflineMsg implements the Channel SendOfflineMsg method.
func (c *InnerChannel) SendOfflineMsg(conn net.Conn, mid int64, key string) error {
	// WARN: inner store must lock
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// find the next node
	for n := c.message.Greate(mid); n != nil; n = n.Next() {
		m, ok := n.Member.(*Message)
		if !ok {
			// never happen
			Log.Crit("skiplist memeber assection type failed")
			panic(AssertTypeErr)
		}

		// check message expired
		if m.Expired() {
			// WARN:though the node deleted, can access the next node
			c.message.Delete(n.Score)
			Log.Warn("user_key:\"%s\" delete the expired message:%d", key, n.Score)
		} else {
			b, err := m.Bytes(nil)
			if err != nil {
				Log.Error("message.Bytes(nil) failed (%s)", err.Error())
				return err
			}

			if _, err := conn.Write(b); err != nil {
				Log.Error("conn.Write() failed (%s)", err.Error())
				return err
			}
		}
	}

	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *InnerChannel) PushMsg(m *Message, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// check message expired
	if m.Expired() {
		Log.Warn("user_key:\"%s\" message:%d has already expired", key, m.MsgID)
		return MsgExpiredErr
	}

	// check exceed the max message length
	if c.message.Length+1 > c.maxMessage {
		// remove the first node cause that's the smallest node
		n := c.message.Head.Next()
		if n == nil {
			// never happen
			Log.Crit("the subscriber touch a impossiable place")
			panic("Skiplist head nil")
		}

		c.message.Delete(n.Score)
		Log.Warn("user_key:\"%s\" message:%d exceed the max message (%d) setting, trim the subscriber", key, n.Score, c.maxMessage)
	}

	err := c.message.Insert(m.MsgID, m)
	if err != nil {
		return err
	}

	b, err := m.Bytes(nil)
	if err != nil {
		Log.Error("message.Bytes(nil) failed (%s)", err.Error())
		return err
	}

	// send message to all the clients
	for conn, _ := range c.conn {
		if _, err = conn.Write(b); err != nil {
			Log.Error("message write error, conn.Write() failed (%s)", err.Error())
			continue
		}

		Log.Info("user_key:\"%s\" push message \"%s\":%d", key, m.Msg, m.MsgID)
	}

	return nil
}

// AddConn implements the Channel AddConn method.
func (c *InnerChannel) AddConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	// check exceed the maxsubscribers
	if len(c.conn)+1 > Conf.MaxSubscriberPerKey {
		c.mutex.Unlock()
		Log.Warn("user_key:\"%s\" exceed the max subscribers", key)
		return MaxConnErr
	}

	c.conn[conn] = true
	c.mutex.Unlock()
	Log.Error("user_key:\"%s\" add conn", key)
	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *InnerChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	delete(c.conn, conn)
	c.mutex.Unlock()
	Log.Error("user_key:\"%s\" remove conn", key)
	return nil
}

// SetDeadline implements the Channel SetDeadline method.
func (c *InnerChannel) SetDeadline(d int64) {
	c.expire = d
}

// Timeout implements the Channel Timeout method.
func (c *InnerChannel) Timeout() bool {
	return time.Now().UnixNano() > c.expire
}

// Close implements the Channel Close method.
func (c *InnerChannel) Close() error {
	c.mutex.Lock()
	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Error("conn.Close() failed (%s)", err.Error())
		}
	}

	c.mutex.Unlock()
	return nil
}
