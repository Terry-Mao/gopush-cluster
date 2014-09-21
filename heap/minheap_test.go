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

package heap

import (
	"fmt"
	"testing"
)

func Test(t *testing.T) {
	h := NewMinheap(2)
	// add
	fmt.Println("------------- add ----------------")
	h.Add(&Element{Key: 20, Value: 1})
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	h.Add(&Element{Key: 15, Value: 1})
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	h.Add(&Element{Key: 2, Value: 1})
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	h.Add(&Element{Key: 14, Value: 1})
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	h.Add(&Element{Key: 10, Value: 1})
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	// poll
	fmt.Println("------------- poll ----------------")
	e := h.Poll()
	fmt.Printf("FETCH Key: %d, Value: %d\n", e.Key, e.Value)
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	e = h.Poll()
	fmt.Printf("FETCH Key: %d, Value: %d\n", e.Key, e.Value)
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	e = h.Poll()
	fmt.Printf("FETCH Key: %d, Value: %d\n", e.Key, e.Value)
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	e = h.Poll()
	fmt.Printf("FETCH Key: %d, Value: %d\n", e.Key, e.Value)
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
	e = h.Poll()
	fmt.Printf("FETCH Key: %d, Value: %d\n", e.Key, e.Value)
	fmt.Printf("Size: %d, Max: %d\n", h.Size(), h.Max())
	for i := 0; i < h.Size(); i++ {
		fmt.Printf("Key: %d, Value: %d\n", h.items[i].Key, h.items[i].Value)
	}
}
