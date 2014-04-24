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
	"errors"
	"github.com/golang/glog"
	"time"
)

const (
	TCPProto               = uint8(0)
	WebsocketProto         = uint8(1)
	WebsocketProtoStr      = "websocket"
	TCPProtoStr            = "tcp"
	Heartbeat              = "h"
	minHearbeatSec         = 30
	delayHeartbeatSec      = 5
	fitstPacketTimedoutSec = time.Second * 5
	Second                 = int64(time.Second)
)

var (
	// Exceed the max subscriber per key
	ErrMaxConn = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	ErrAssertType = errors.New("Subscriber assert type failed")
	// Heartbeat
	// HeartbeatLen = len(Heartbeat)
	// hearbeat
	HeartbeatReply = []byte("+h\r\n")
	// auth failed reply
	AuthReply = []byte("-a\r\n")
	// channle not found reply
	ChannelReply = []byte("-c\r\n")
	// param error reply
	ParamReply = []byte("-p\r\n")
)

// StartListen start accept client.
func StartComet() {
	for _, proto := range Conf.Proto {
		if proto == WebsocketProtoStr {
			// Start http push service
			StartHttp()
		} else if proto == TCPProtoStr {
			// Start tcp push service
			StartTCP()
		} else {
			glog.Warningf("unknown gopush-cluster protocol %s, (\"websocket\" or \"tcp\")", proto)
		}
	}
}
