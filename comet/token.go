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
		Log.Warn("token \"%s\" exist", ticket)
		return ErrTokenExist
	}
	t.clean()
	return nil
}

// Auth auth a token is valid
func (t *Token) Auth(ticket string) error {
	if e, ok := t.token[ticket]; !ok {
		Log.Warn("token \"%s\" not exist", ticket)
		return ErrTokenNotExist
	} else {
		td, _ := e.Value.(*TokenData)
		if time.Now().After(td.Expire) {
			t.clean()
			Log.Warn("token \"%s\" expired", ticket)
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
			Log.Warn("token \"%s\" expired", td.Ticket)
			o := e.Next()
			delete(t.token, td.Ticket)
			t.lru.Remove(e)
			e = o
			continue
		}
		break
	}
}
