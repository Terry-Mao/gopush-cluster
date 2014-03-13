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

package log

import (
	"testing"
)

// create a log in /tmp/test.log
func TestLog(t *testing.T) {
	logger, err := New("/tmp/test.log", "ERROR")
	if err != nil {
		logger.Error("write to stdout")
		t.Error("new a logger failed")
	}
	logger.Error("come on %d", 111)
	logger.Info("won't in log")
	if err = logger.Close(); err != nil {
		t.Error("close a logger failed")
	}
	logger, err = New("", "ERROR")
	if err != nil {
		logger.Error("write to stdout")
		t.Error("new a logger failed")
	}
	logger.Error("write to stdout, %s", "test")
}
