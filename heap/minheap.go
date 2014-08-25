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

// Element is an element of a Minheap.
type Element struct {
	// The key stored the heap node key.
	Key int
	// The value stored with this element.
	Value interface{}
}

// Minheap is priority queue(min heap) struct.
type Minheap struct {
	size  int
	max   int
	items []*Element
}

// NewMinheap new a Minheap struct with specified size.
func NewMinheap(max int) *Minheap {
	return &Minheap{size: 0, max: max, items: make([]*Element, max, max)}
}

// full check the minheap is full.
func (h *Minheap) full() bool {
	return h.size >= h.max
}

// empty check the minheap is empty.
func (h *Minheap) empty() bool {
	return h.size <= 0
}

// Min get the top of Minheap.
func (h *Minheap) Min() *Element {
	if h.empty() {
		return nil
	} else {
		return h.items[0]
	}
}

// Add add a element to minheap.
func (h *Minheap) Add(e *Element) {
	if e == nil {
		return
	}
	if h.full() {
		h.grow()
	}
	s := h.size
	h.size++
	for {
		if s <= 0 {
			break
		}
		p := (s - 1) / 2
		if h.items[p].Key < e.Key {
			break
		}
		// parent shift down
		h.items[s] = h.items[p]
		s = p
	}
	h.items[s] = e
}

// grow grow the minheap max size.
func (h *Minheap) grow() {
	ns := h.max * 2
	ni := make([]*Element, ns, ns)
	copy(ni, h.items)
	h.items = ni
	h.max = ns
}

// Poll remove top element in minheap.
func (h *Minheap) Poll() *Element {
	if h.empty() {
		return nil
	}
	h.size--
	// save value of the element with the highest key
	e := h.items[0]
	// use last element in heap to adjust heap
	t := h.items[h.size]
	i := 0
	for {
		half := h.size / 2
		if i >= half {
			break
		}
		c := 2*i + 1
		r := c + 1
		// use left or right node
		if r < h.size && h.items[c].Key > h.items[r].Key {
			c = r
		}
		if t.Key < h.items[c].Key {
			break
		}
		// child shift up
		h.items[i] = h.items[c]
		i = c
	}
	h.items[i] = t
	h.items[h.size] = nil
	return e
}

// Size get minheap current size
func (h *Minheap) Size() int {
	return h.size
}

// Max get minheap max size
func (h *Minheap) Max() int {
	return h.max
}
