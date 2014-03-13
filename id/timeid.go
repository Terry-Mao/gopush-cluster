package id

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
			// if last time id > current time, may be who change the system id,
			// so sleep last time id minus current time
			time.Sleep(time.Duration((t.lastID-s+1)*100) * time.Nanosecond)
		} else {
			// save the current time id
			t.lastID = s
			return s
		}
	}
	return 0
}
