package main

import (
	"errors"
	"fmt"
	"time"
)

const (
	WebsocketProtocol = 0
	TCPProtocol       = 1
	Heartbeat         = "h"
	oneSecond         = int64(time.Second)
)

var (
	// Exceed the max subscriber per key
	MaxConnErr = errors.New("Exceed the max subscriber connection per key")
	// Assection type failed
	AssertTypeErr = errors.New("Subscriber assert type failed")

	// Heartbeat
	HeartbeatLen = len(Heartbeat)
	// websocket heartbeat
	WebsocketHeartbeatReply = []byte(Heartbeat)
	// tcp hearbeat
	TCPHeartbeatReply = []byte(fmt.Sprintf("$1\r\n%s\r\n", Heartbeat))
)
