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

const (
	PrivateGroupId = 0
	PublicGroupId  = 1
)

// Channel Push Private Message Args
type ChannelPushPrivateArgs struct {
	Key     string          // subscriber key
	Msg     json.RawMessage // message content
	GroupId uint            // message group id
	Expire  uint            // message expire second
}

// Channel Push Public Message Args
type ChannelPushPublicArgs struct {
	MsgID int64  // message id
	Msg   string // message content
}

// Channel Migrate Args
type ChannelMigrateArgs struct {
	Nodes []string // current comet nodes
	Vnode int      // ketama virtual node number
}

// Channel New Args
type ChannelNewArgs struct {
	Expire int64  // message expire second
	Token  string // auth token
	Key    string // subscriber key
}
