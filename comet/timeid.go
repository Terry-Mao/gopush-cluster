package main

import (
	"time"
)

type TimeID struct {
	lastID int64
}

// NewTimeID create a new TimeID struct
func NewTimeID() *TimeID {
	return &TimeID{lastID: 0}
}

// ID generate a time ID
func (t *TimeID) ID() int64 {
	for {
		s := time.Now().UnixNano() / 100
		if t.lastID >= s {
			time.Sleep(100 * time.Nanosecond)
		} else {
			t.lastID = s
			return s
		}
	}
	return 0
}
