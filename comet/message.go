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
