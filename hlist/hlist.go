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

package hlist

// Head is an head of a linked hlist.
type Head struct {
	first *Element
}

// Element is an element of a linked hlist.
type Element struct {
	next  *Element
	pprev **Element

	// The value stored with this element.
	Value interface{}
}

// Next returns the next hlist element or nil.
func (e *Element) Next() *Element {
	return e.next
}

// Hlist represents a doubly linked hlist.
// The zero value for Hlist is an empty Hlist ready to use.
type Hlist struct {
	root Head // sentinel hlist head
	len  int  // current hlist length excluding (this) sentinel element
}

// Init initializes or clears hlist l.
func (l *Hlist) Init() *Hlist {
	l.root.first = nil
	l.len = 0
	return l
}

// New returns an initialized hlist.
func New() *Hlist { return new(Hlist).Init() }

// Len returns the number of elements of hlist l.
// The complexity is O(1).
func (l *Hlist) Len() int { return l.len }

// Front returns the first element of hlist l or nil
func (l *Hlist) Front() *Element {
	return l.root.first
}

// PushFront inserts a new element e with value v at the front of hlist l and returns e.
func (l *Hlist) PushFront(v interface{}) *Element {
	first := l.root.first
	n := &Element{Value: v}
	n.next = first
	if first != nil {
		first.pprev = &n.next
	}
	l.root.first = n
	n.pprev = &l.root.first
	l.len++
	return n
}

// Remove removes e from l if e is an element of hlist l.
// It returns the element value e.Value.
func (l *Hlist) Remove(e *Element) interface{} {
	next := e.next
	pprev := e.pprev
	*pprev = next
	if next != nil {
		next.pprev = pprev
	}
	l.len--
	e.next = nil  // avoid memory leak
	e.pprev = nil // avoid memory leak
	return e.Value
}
