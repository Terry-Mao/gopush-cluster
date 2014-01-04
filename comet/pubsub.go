package main

import (
	"errors"
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
)
