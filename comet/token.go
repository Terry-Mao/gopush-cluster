package main

import (
	"container/list"
	"errors"
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
	Expire int64
}

// NewToken create a token struct ptr
func NewToken() *Token {
	return &Token{
		token: map[string]*list.Element{},
		lru:   list.New(),
	}
}

// Add add a token
func (t *Token) Add(ticket string, expire int64) error {
	if e, ok := t.token[ticket]; !ok {
		// new element add to lru back
		e = t.lru.PushBack(&TokenData{Ticket: ticket, Expire: expire})
		t.token[ticket] = e
	} else {
		Log.Error("add token %s:%d exist", ticket, expire)
		return ErrTokenExist
	}

	t.clean()
	return nil
}

// Auth auth a token is valid
func (t *Token) Auth(ticket string) error {
	if e, ok := t.token[ticket]; !ok {
		Log.Error("auth token %s not exist", ticket)
		return ErrTokenNotExist
	} else {
		td, _ := e.Value.(*TokenData)
		if td.Expire < time.Now().UnixNano() {
			Log.Error("token %s expired", ticket)
			delete(t.token, td.Ticket)
			return ErrTokenExpired
		}
	}

	t.clean()
	return nil
}

// Expire set the expired time for the ticket
func (t *Token) Expire(ticket string, expire int64) error {
	if e, ok := t.token[ticket]; !ok {
		Log.Error("expire token %s not exist", ticket)
		return ErrTokenNotExist
	} else {
		// refresh ttl
		td, _ := e.Value.(*TokenData)
		td.Expire = expire
		t.lru.MoveToBack(e)
	}

	return nil
}

// expire scan the lru list expire the element
func (t *Token) clean() {
	now := time.Now().UnixNano()
	e := t.lru.Front()
	for {
		if e == nil {
			break
		}

		td, _ := e.Value.(*TokenData)
		if td.Expire < now {
			Log.Warn("token %s:%d expired", td.Ticket, td.Expire)
			o := e.Next()
			t.lru.Remove(e)
			e = o
			delete(t.token, td.Ticket)
			continue
		} else {
			break
		}
	}
}
