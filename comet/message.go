package main

import (
	"encoding/json"
	"fmt"
)

// The Message struct.
type Message struct {
	// Message expired time
	Expire int64 `json:"-"`
	// Message
	Msg string `json:"msg"`
	// Message id
	MsgID int64 `json:"mid"`
	// Group id
	GroupID int `json:"gid"`
}

// Bytes get a message reply bytes.
func (m *Message) Bytes() ([]byte, error) {
	byteJson, err := json.Marshal(m)
	if err != nil {
		Log.Error("json.Marshal(%v) failed (%s)", m, err.Error())
		return nil, err
	}
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(byteJson), string(byteJson))), nil
}
