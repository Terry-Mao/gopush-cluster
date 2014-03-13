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
