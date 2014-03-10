package main

import (
	"encoding/json"
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
		Log.Error("json.Marshal(%v) error(%v)", m, err)
		return nil, err
	}

	return byteJson, nil
}
