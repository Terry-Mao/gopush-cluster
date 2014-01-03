package main

import (
	"bytes"
	"errors"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/Terry-Mao/gopush-cluster/skiplist"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	snowflakeMachineID    = int64(0)
	messageSetDialTimeout = 5
	messageSetDeadline    = 3
)

var (
	MessageSaveErr = errors.New("Message set failed")
	httpTransport  = &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			deadline := time.Now().Add(messageSetDialTimeout * time.Second)
			c, err := net.DialTimeout(netw, addr, messageSetDeadline*time.Second)
			if err != nil {
				return nil, err
			}

			c.SetDeadline(deadline)
			return c, nil
		},

		DisableKeepAlives: false,
	}

	httpClient = &http.Client{
		Transport: httpTransport,
	}
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
	// snowflake lastSID
	lastSID int64
	// snowflake lastTtime
	lastTime int64
	// token expire
	tokenTTL *skiplist.SkipList
	// token
	token map[string]int64
}

// snlwflakeID generate a time-based id for message
func (c *OuterChannel) snowflakeID() int64 {
	sid := int64(0)
	t := time.Now().UnixNano() / 1000000
	if t == c.lastTime {
		sid = c.lastSID + 1
		//sid必须在[0,4095],如果不在此范围，则sleep 1 毫秒，进入下一个时间点。
		if sid > 4095 || sid < 0 {
			time.Sleep(1 * time.Millisecond)
			t = time.Now().UnixNano() / 1000000
			sid = 0
		}
	}

	c.lastTime = t
	c.lastSID = sid
	return (t << 22) + sid<<10 + snowflakeMachineID
}

// New a user outer stored message channel
func NewOuterChannel() *OuterChannel {
	return &OuterChannel{
		mutex:    &sync.Mutex{},
		conn:     map[net.Conn]int64{},
		expire:   time.Now().UnixNano() + Conf.ChannelExpireSec*Second,
		writeBuf: make(chan *bytes.Buffer, Conf.WriteBufNum),
		lastSID:  0,
		lastTime: 0,
		token:    map[string]int64{},
		tokenTTL: skiplist.New(),
	}
}

// newWriteBuf get a buf from channel or create a new buf if chan is empty
func (c *OuterChannel) newWriteBuf() *bytes.Buffer {
	select {
	case buf := <-c.writeBuf:
		buf.Reset()
		return buf
	default:
		Log.Debug("outer channel buffer empty")
		return bytes.NewBuffer(make([]byte, 0, Conf.WriteBufByte))
	}
}

// pubWriteBuf return back a buf, if chan is full then discard
func (c *OuterChannel) putWriteBuf(buf *bytes.Buffer) {
	select {
	case c.writeBuf <- buf:
	default:
		Log.Debug("outer channel buffer full")
	}
}

// SendOfflineMsg implements the Channel SendOfflineMsg method.
func (c *OuterChannel) SendOfflineMsg(conn net.Conn, mid int64, key string) error {
	// no need
	return nil
}

// expireToken delete the expired token
func (c *OuterChannel) expireToken(key string) error {
	now := time.Now().UnixNano()
	// find the min node, if expired then continue
	for n := c.tokenTTL.Head.Next(); n != nil; n = n.Next() {
		if n.Score < now {
			if en := c.tokenTTL.Delete(n.Score); en == nil {
				Log.Error("user_key:\"%s\" skiplist delete node %d failed (%s)", key, n.Score)
				Log.Crit("user_key:\"%s\" skiplist linked list fatal error", key)
				return TokenDeleteErr
			}

			m, _ := n.Member.(string)
			delete(c.token, m)
		} else {
			Log.Debug("user_key:\"%s\" skiplist no other node expired", key)
			break
		}
	}

	return nil
}

// AddToken implements the Channel AddToken method.
func (c *OuterChannel) AddToken(token string, expire int64, key string) error {
	if expire <= time.Now().UnixNano() {
		Log.Error("user_key:\"%s\" add token %s already expired", key, token)
		return TokenExpiredErr
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.token[token]; ok {
		Log.Error("user_key:\"%s\" add token %s already exists", key, token)
		return TokenExistErr
	} else {
		c.token[token] = expire
		if err := c.tokenTTL.Insert(expire, token); err != nil {
			Log.Error("user_key:\"%s\" skiplist token %s insert failed (%s)", key, token, err.Error())
			return err
		}
	}

	// check has expired token
	if err := c.expireToken(key); err != nil {
		Log.Error("user_key:\"%s\" expire token failed (%s)", key, err.Error())
	}

	return nil
}

// AuthToken implements the Channel AuthToken method.
func (c *OuterChannel) AuthToken(token string, key string) error {
	now := time.Now().UnixNano()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// token only used once
	if t, ok := c.token[token]; ok {
		if dn := c.tokenTTL.Delete(t); dn == nil {
			Log.Error("user_key:\"%s\" skiplist delete node %d failed", key, t)
			return TokenDeleteErr
		}

		delete(c.token, token)
		if t < now {
			Log.Error("user_key:\"%s\" add token %s already expired", key, token)
			return TokenExpiredErr
		}
	} else {
		Log.Error("user_key:\"%s\" auth token %s not exists", key, token)
		return TokenNotExistErr
	}

	// check has expired token
	if err := c.expireToken(key); err != nil {
		Log.Error("user_key:\"%s\" expire token failed (%s)", key, err.Error())
	}

	return nil
}

// PushMsg implements the Channel PushMsg method.
func (c *OuterChannel) PushMsg(m *Message, key string) error {
	// fetch a write buf, return back after call end
	buf := c.newWriteBuf()
	defer c.putWriteBuf(buf)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// rewrite message id
	m.MsgID = c.snowflakeID()
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
	b, err := m.Bytes(buf)
	if err != nil {
		Log.Error("message.Bytes(buf) failed (%s)", err.Error())
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
