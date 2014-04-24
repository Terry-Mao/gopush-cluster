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

package main

import (
	"container/list"
	"errors"
	"github.com/golang/glog"
	"time"
)

var (
	// Token exists
	ErrTokenExist = errors.New("token exist")
	// Token not exists
	ErrTokenNotExist = errors.New("token not exist")
	// Token expired
	ErrTokenExpired = errors.New("token expired")
)

// Token struct
type Token struct {
	token map[string]*list.Element // token map
	lru   *list.List               // lru double linked list
}

// Token Element
type TokenData struct {
	Ticket string
	Expire time.Time
}

// NewToken create a token struct ptr
func NewToken() *Token {
	return &Token{
		token: map[string]*list.Element{},
		lru:   list.New(),
	}
}

// Add add a token
func (t *Token) Add(ticket string) error {
	if e, ok := t.token[ticket]; !ok {
		// new element add to lru back
		e = t.lru.PushBack(&TokenData{Ticket: ticket, Expire: time.Now().Add(Conf.TokenExpire)})
		t.token[ticket] = e
	} else {
		glog.Warningf("token \"%s\" exist", ticket)
		return ErrTokenExist
	}
	t.clean()
	return nil
}

// Auth auth a token is valid
func (t *Token) Auth(ticket string) error {
	if e, ok := t.token[ticket]; !ok {
		glog.Warningf("token \"%s\" not exist", ticket)
		return ErrTokenNotExist
	} else {
		td, _ := e.Value.(*TokenData)
		if time.Now().After(td.Expire) {
			t.clean()
			glog.Warningf("token \"%s\" expired", ticket)
			return ErrTokenExpired
		}
		td.Expire = time.Now().Add(Conf.TokenExpire)
		t.lru.MoveToBack(e)
	}
	t.clean()
	return nil
}

// clean scan the lru list expire the element
func (t *Token) clean() {
	now := time.Now()
	e := t.lru.Front()
	for {
		if e == nil {
			break
		}
		td, _ := e.Value.(*TokenData)
		if now.After(td.Expire) {
			glog.Warningf("token \"%s\" expired", td.Ticket)
			o := e.Next()
			delete(t.token, td.Ticket)
			t.lru.Remove(e)
			e = o
			continue
		}
		break
	}
}
