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

package hash

import (
	"testing"
)

func TestKetama(t *testing.T) {
	k := NewKetama(15, 255)
	n := k.Node("Terry-Mao332")
	if n != "node15" {
		t.Error("Terry-Mao332 must hit node13")
	}

	n = k.Node("Terry-Mao2")
	if n != "node13" {
		t.Error("Terry-Mao2 must hit node13")
	}

	n = k.Node("Terry-Mao3")
	if n != "node5" {
		t.Error("Terry-Mao3 must hit node13")
	}

	n = k.Node("Terry-Mao4")
	if n != "node2" {
		t.Error("Terry-Mao4 must hit node13")
	}

	n = k.Node("Terry-Mao5")
	if n != "node6" {
		t.Error("Terry-Mao5 must hit node13")
	}
}

func TestKetama2(t *testing.T) {
	k := NewKetama2([]string{"11", "22"}, 255)
	n := k.Node("Terry-Mao332")
	if n != "22" {
		t.Error("Terry-Mao332 must hit 22")
	}

	n = k.Node("Terry-Mao333")
	if n != "22" {
		t.Error("Terry-Mao333 must hit 22")
	}

	n = k.Node("Terry-Mao334")
	if n != "22" {
		t.Error("Terry-Mao333 must hit 22")
	}

	n = k.Node("Terry-Mao335")
	if n != "11" {
		t.Error("Terry-Mao335 must hit 11")
	}
}
