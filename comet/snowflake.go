package main

import (
	"time"
)

const (
	machineID = int64(0)
)

// Snowflake a id generator struct
type Snowflake struct {
	// snowflake lastSID
	lastSID int64
	// snowflake lastTtime
	lastTime int64
}

// NewSnowflake create a new snowflake struct
func NewSnowflake() *Snowflake {
	return &Snowflake{lastSID: 0, lastTime: 0}
}

// ID generate a snowflake ID
func (s *Snowflake) ID() int64 {
	sid := int64(0)
	t := time.Now().UnixNano() / 1000000
	if t == s.lastTime {
		sid = s.lastSID + 1
		//sid必须在[0,4095],如果不在此范围，则sleep 1 毫秒，进入下一个时间点。
		if sid > 4095 || sid < 0 {
			time.Sleep(1 * time.Millisecond)
			t = time.Now().UnixNano() / 1000000
			sid = 0
		}
	}

	s.lastTime = t
	s.lastSID = sid
	return (t << 22) + sid<<10 + machineID
}
