package main

import (
	"bytes"
	"net"
	"sync"
	"time"
)

type OuterChannel struct {
	// Mutex
	mutex *sync.Mutex
	// Client conn
	conn map[net.Conn]int64
	// Channel expired unixnano
	expire int64
	// write buffer chan for message sent
	writeBuf chan *bytes.Buffer
}

// New a user outer stored message channel
func NewOuterChannel() *OuterChannel {
	return &OuterChannel{
		mutex:    &sync.Mutex{},
		conn:     map[net.Conn]int64{},
		expire:   time.Now().UnixNano() + Conf.ChannelExpireSec*Second,
		writeBuf: make(chan *bytes.Buffer, Conf.WriteBufNum),
	}
}

// newWriteBuf get a buf from channel or create a new buf if chan is empty
func (c *OuterChannel) newWriteBuf() *bytes.Buffer {
	select {
	case buf := <-c.writeBuf:
		buf.Reset()
		return buf
	default:
		return bytes.NewBuffer(make([]byte, Conf.WriteBufByte))
	}
}

// pubWriteBuf return back a buf, if chan is full then discard
func (c *OuterChannel) putWriteBuf(buf *bytes.Buffer) {
	select {
	case c.writeBuf <- buf:
	default:
	}
}

// SendOfflineMsg implements the Channel SendOfflineMsg method.
func (c *OuterChannel) SendOfflineMsg(conn net.Conn, mid int64, key string) error {
	// no need
	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *OuterChannel) PushMsg(m *Message, key string) error {
	// fetch a write buf, return back after call end
	buf := c.newWriteBuf()
	defer c.putWriteBuf(buf)
	// send message to each conn when message id > conn last message id
	b, err := m.Bytes(buf)
	if err != nil {
		Log.Error("message.Bytes(buf) failed (%s)", err.Error())
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	for conn, mid := range c.conn {
		// ignore message cause it's id less than mid
		if mid >= m.MsgID {
			Log.Warn("user_key:\"%s\" ignore mid:%d, last mid:%d", key, m.MsgID, mid)
			continue
		}

		if _, err = conn.Write(b); err != nil {
			Log.Error("conn.Write() failed (%s)", err.Error())
			continue
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
	Log.Info("user_key:\"%s\" add conn", key)
	return nil
}

// RemoveConn implements the Channel RemoveConn method.
func (c *OuterChannel) RemoveConn(conn net.Conn, mid int64, key string) error {
	c.mutex.Lock()
	delete(c.conn, conn)
	c.mutex.Unlock()
	Log.Error("user_key:\"%s\" remove conn", key)
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
	c.mutex.Lock()

	for conn, _ := range c.conn {
		if err := conn.Close(); err != nil {
			// ignore close error
			Log.Warn("conn.Close() failed (%s)", err.Error())
		}
	}

	c.mutex.Unlock()
	return nil
}
