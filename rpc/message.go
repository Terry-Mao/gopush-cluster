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

package rpc

import (
    "encoding/json"
)

// Message Save Args
type MessageSaveArgs struct {
	Key    string // subscriber key
	Msg    json.RawMessage // message content
    MsgId   int64 // message id
	GroupId int    // message group id
	Expire int64  // message expire second
}

// Public Message Save Args
type MessageSavePubArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
}

// Message Get Args
type MessageGetArgs struct {
	MsgID    int64  // message id
	PubMsgID int64  // public message id
	Key      string // subscriber key
}

// Message Get Response
type MessageGetResp struct {
	Ret     int      // response
	Msgs    []string // messages
	PubMsgs []string // public messages
}
