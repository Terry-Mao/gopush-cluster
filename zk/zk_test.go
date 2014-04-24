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

// github.com/samuel/go-zookeeper
// Copyright (c) 2013, Samuel Stauffer <samuel@descolada.com>
// All rights reserved.

package zk

import (
	"testing"
	"time"
)

func TestZK(t *testing.T) {
	conn, err := Connect([]string{"10.33.21.152:2181"}, time.Second*30)
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()
	err = Create(conn, "/test/test")
	if err != nil {
		t.Error(err)
	}
	// registertmp
	err = RegisterTemp(conn, "/test/test", "1")
	if err != nil {
		t.Error(err)
	}
}
