// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"github.com/golang/glog"
)

// The Message struct.
type Message struct {
	// Message
	Msg json.RawMessage `json:"msg"`
	// Message id
	MsgId int64 `json:"mid"`
	// Group id
	GroupId uint `json:"gid"`
}

// The Old Message struct (Compatible), TODO remove it.
type OldMessage struct {
	// Message
	Msg string `json:"msg"`
	// Message id
	MsgId int64 `json:"mid"`
	// Group id
	GroupId uint `json:"gid"`
}

// Bytes get a message reply bytes.
func (m *Message) Bytes() ([]byte, error) {
	byteJson, err := json.Marshal(m)
	if err != nil {
		glog.Errorf("json.Marshal(%v) error(%v)", m, err)
		return nil, err
	}
	return byteJson, nil
}

// OldBytes get a message reply bytes(Compatible), TODO remove it.
func (m *Message) OldBytes() ([]byte, error) {
	om := &OldMessage{Msg: string(m.Msg), MsgId: m.MsgId, GroupId: m.GroupId}
	byteJson, err := json.Marshal(om)
	if err != nil {
		glog.Errorf("json.Marshal(%v) error(%v)", om, err)
		return nil, err
	}
	return byteJson, nil
}
