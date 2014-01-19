package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrMsgExpired = errors.New("Message already expired")
)

// The Message struct.
type Message struct {
	// Message
	Msg string `json:"msg"`
	// Message expired unixnano
	Expire int64 `json:"expire"`
	// Message id
	MsgID int64 `json:"mid"`
	// Group id
	GroupID int `json:"gid"`
}

// Expired check mesage expired or not.
func (m *Message) Expired() bool {
	return time.Now().UnixNano() > m.Expire
}

// NewJsonStrMessage unmarshal the Message.
func NewJsonStrMessage(str string) (*Message, error) {
	m := &Message{}
	err := json.Unmarshal([]byte(str), m)
	if err != nil {
		Log.Error("json.Unmarshal() failed (%s), message json: \"%s\"", err.Error(), str)
		return nil, err
	}

	return m, nil
}

// Bytes get a message bytes.
func (m *Message) Bytes() ([]byte, error) {
	res := map[string]interface{}{
		"msg": m.Msg,
		"mid": m.MsgID,
		"gid": m.GroupID,
	}

	byteJson, err := json.Marshal(res)
	if err != nil {
		Log.Error("json.Marshal(%v) failed (%s)", res, err.Error())
		return nil, err
	}

	// tcp use Redis Response Protocol: http://redis.io/topics/protocol
	if Conf.Protocol == TCPProtocol {
		return messageReply(byteJson), nil
	} else {
		// websocket
		return byteJson, nil
	}
}

// redis protocol reply
func messageReply(msg []byte) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(msg), string(msg)))
}
