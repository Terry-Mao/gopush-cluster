package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

var (
	// Message expired
	MsgExpiredErr = errors.New("Message already expired")
	MsgBufErr     = errors.New("Message writeu buffer nil")
)

// The Message struct
type Message struct {
	// Message
	Msg string `json:"msg"`
	// Message expired unixnano
	Expire int64 `json:"expire"`
	// Message id
	MsgID int64 `json:"mid"`
}

// Expired check mesage expired or not
func (m *Message) Expired() bool {
	return time.Now().UnixNano() > m.Expire
}

// NewJsonStrMessage unmarshal the Message
func NewJsonStrMessage(str string) (*Message, error) {
	m := &Message{}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		Log.Error("json.Unmarshal() failed (%s), message json: \"%s\"", err.Error(), str)
		return nil, err
	}

	return m, nil
}

// Bytes get a message bytes
func (m *Message) Bytes(b *bytes.Buffer) ([]byte, error) {
	res := map[string]interface{}{
		"msg": m.Msg,
		"mid": m.MsgID,
	}

	byteJson, err := json.Marshal(res)
	if err != nil {
		Log.Error("json.Marshal(%v) failed (%s)", res, err.Error())
		return nil, err
	}

	// tcp use Redis Response Protocol: http://redis.io/topics/protocol
	if Conf.Protocol == TCPProtocol {
		if b == nil {
			return nil, MsgBufErr
		}

		// $size\r\ndata\r\n
		if _, err = b.WriteString(fmt.Sprintf("$%d\r\n", len(byteJson))); err != nil {
			Log.Error("buf.WriteString() failed (%s)", err.Error())
			return nil, err
		}

		if _, err = b.Write(byteJson); err != nil {
			Log.Error("buf.Write() failed (%s)", err.Error())
			return nil, err
		}

		if _, err = b.WriteString("\r\n"); err != nil {
			Log.Error("buf.WriteString() failed (%s)", err.Error())
			return nil, err
		}

		return b.Bytes(), nil
	} else {
		// websocket
		return byteJson, nil
	}
}

func (m *Message) PostString(key string) string {
	d := url.Values{}
	d.Set("msg_id", strconv.FormatInt(m.MsgID, 10))
	d.Set("expire", strconv.FormatInt(m.Expire, 10))
	d.Set("msg", m.Msg)
	d.Set("key", key)
	return d.Encode()
}
