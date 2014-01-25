package main

import (
	"errors"
	"strings"
	"time"
)

const (
	WebsocketProtocol = "websocket"
	TCPProtocol       = "tcp"
	Heartbeat         = "h"
	minHearbeatSec    = 30
	Second            = int64(time.Second)
)

var (
	// Exceed the max subscriber per key
	ErrMaxConn = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	ErrAssertType = errors.New("Subscriber assert type failed")
	// Heartbeat
	HeartbeatLen = len(Heartbeat)
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
	for _, proto := range strings.Split(Conf.Proto, ",") {
		proto = strings.TrimSpace(proto)
		if proto == WebsocketProtocol {
			// Start http push service
			go StartHttp()
		} else if proto == TCPProtocol {
			// Start http push service
			go StartTCP()
		} else {
			Log.Warn("unknown gopush-cluster protocol %s, (\"websocket\" or \"tcp\")", proto)
		}
	}
}
