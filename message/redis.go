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
	log "code.google.com/p/log4go"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/ketama"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/garyburd/redigo/redis"
	"strconv"
	"strings"
	"time"
)

var (
	RedisNoConnErr = errors.New("can't get a redis conn")
)

// RedisMessage struct encoding the composite info.
type RedisPrivateMessage struct {
	Msg    json.RawMessage `json:"msg"`    // message content
	Expire int64           `json:"expire"` // expire second
}

// Struct for delele message
type RedisDelMessage struct {
	Key  string
	MIds []int64
}

type RedisStorage struct {
	pool  map[string]*redis.Pool
	ring  *ketama.HashRing
	delCH chan *RedisDelMessage
}

// NewRedis initialize the redis pool and consistency hash ring.
func NewRedisStorage() *RedisStorage {
	var (
		err error
		w   int
		nw  []string
	)
	redisPool := map[string]*redis.Pool{}
	ring := ketama.NewRing(Conf.RedisKetamaBase)
	for n, addr := range Conf.RedisSource {
		nw = strings.Split(n, ":")
		if len(nw) != 2 {
			err = errors.New("node config error, it's nodeN:W")
			log.Error("strings.Split(\"%s\", :) failed (%v)", n, err)
			panic(err)
		}
		w, err = strconv.Atoi(nw[1])
		if err != nil {
			log.Error("strconv.Atoi(\"%s\") failed (%v)", nw[1], err)
			panic(err)
		}
		tmp := addr
		// WARN: closures use
		redisPool[nw[0]] = &redis.Pool{
			MaxIdle:     Conf.RedisMaxIdle,
			MaxActive:   Conf.RedisMaxActive,
			IdleTimeout: Conf.RedisIdleTimeout,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial("tcp", tmp)
				if err != nil {
					log.Error("redis.Dial(\"tcp\", \"%s\") error(%v)", tmp, err)
					return nil, err
				}
				return conn, err
			},
		}
		ring.AddNode(nw[0], w)
	}
	ring.Bake()
	s := &RedisStorage{pool: redisPool, ring: ring, delCH: make(chan *RedisDelMessage, 10240)}
	go s.clean()
	return s
}

// SavePrivate implements the Storage SavePrivate method.
func (s *RedisStorage) SavePrivate(key string, msg json.RawMessage, mid int64, expire uint) error {
	conn := s.getConn(key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()
	rm := &RedisPrivateMessage{Msg: msg, Expire: int64(expire) + time.Now().Unix()}
	m, err := json.Marshal(rm)
	if err != nil {
		log.Error("json.Marshal() key:\"%s\" error(%v)", key, err)
		return err
	}
	if err := conn.Send("ZADD", key, mid, m); err != nil {
		log.Error("conn.Send(\"ZADD\", \"%s\", %d, \"%s\") error(%v)", key, mid, string(m), err)
		return err
	}
	if err := conn.Send("ZREMRANGEBYRANK", key, 0, -1*(Conf.RedisMaxStore+1)); err != nil {
		log.Error("conn.Send(\"ZREMRANGEBYRANK\", \"%s\", 0, %d) error(%v)", key, -1*(Conf.RedisMaxStore+1), err)
		return err
	}
	if err := conn.Flush(); err != nil {
		log.Error("conn.Flush() error(%v)", err)
		return err
	}
	_, err = conn.Receive()
	if err != nil {
		log.Error("conn.Receive() error(%v)", err)
		return err
	}
	_, err = conn.Receive()
	if err != nil {
		log.Error("conn.Receive() error(%v)", err)
		return err
	}
	return nil
}

// GetPrivate implements the Storage GetPrivate method.
func (s *RedisStorage) GetPrivate(key string, mid int64) ([]*myrpc.Message, error) {
	conn := s.getConn(key)
	if conn == nil {
		return nil, RedisNoConnErr
	}
	defer conn.Close()
	values, err := redis.Values(conn.Do("ZRANGEBYSCORE", key, fmt.Sprintf("(%d", mid), "+inf", "WITHSCORES"))
	if err != nil {
		log.Error("conn.Do(\"ZRANGEBYSCORE\", \"%s\", \"%d\", \"+inf\", \"WITHSCORES\") error(%v)", key, mid, err)
		return nil, err
	}
	msgs := make([]*myrpc.Message, 0, len(values))
	delMsgs := []int64{}
	now := time.Now().Unix()
	for len(values) > 0 {
		cmid := int64(0)
		b := []byte{}
		values, err = redis.Scan(values, &b, &cmid)
		if err != nil {
			log.Error("redis.Scan() error(%v)", err)
			return nil, err
		}
		rm := &RedisPrivateMessage{}
		if err := json.Unmarshal(b, rm); err != nil {
			log.Error("json.Unmarshal(\"%s\", rm) error(%v)", string(b), err)
			delMsgs = append(delMsgs, cmid)
			continue
		}
		// check expire
		if rm.Expire < now {
			log.Warn("user_key: \"%s\" msg: %d expired", key, cmid)
			delMsgs = append(delMsgs, cmid)
			continue
		}
		m := &myrpc.Message{MsgId: cmid, Msg: rm.Msg, GroupId: myrpc.PrivateGroupId}
		msgs = append(msgs, m)
	}
	// delete unmarshal failed and expired message
	if len(delMsgs) > 0 {
		select {
		case s.delCH <- &RedisDelMessage{Key: key, MIds: delMsgs}:
		default:
			log.Warn("user_key: \"%s\" send del messages failed, channel full", key)
		}
	}
	return msgs, nil
}

// DelPrivate implements the Storage DelPrivate method.
func (s *RedisStorage) DelPrivate(key string) error {
	conn := s.getConn(key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()
	_, err := conn.Do("DEL", key)
	if err != nil {
		log.Error("conn.Do(\"DEL\", \"%s\") error(%v)", key, err)
		return err
	}
	return nil
}

// DelMulti implements the Storage DelMulti method.
func (s *RedisStorage) clean() {
	for {
		info := <-s.delCH
		conn := s.getConn(info.Key)
		if conn == nil {
			log.Warn("get redis connection nil")
			continue
		}
		for _, mid := range info.MIds {
			if err := conn.Send("ZREMRANGEBYSCORE", info.Key, mid, mid); err != nil {
				log.Error("conn.Send(\"ZREMRANGEBYSCORE\", \"%s\", %d, %d) error(%v)", info.Key, mid, mid, err)
				conn.Close()
				continue
			}
		}
		if err := conn.Flush(); err != nil {
			log.Error("conn.Flush() error(%v)", err)
			conn.Close()
			continue
		}
		for _, _ = range info.MIds {
			_, err := conn.Receive()
			if err != nil {
				log.Error("conn.Receive() error(%v)", err)
				conn.Close()
				continue
			}
		}
		conn.Close()
	}
}

// getConn get the connection of matching with key using ketama hashing.
func (s *RedisStorage) getConn(key string) redis.Conn {
	if len(s.pool) == 0 {
		return nil
	}
	node := s.ring.Hash(key)
	p, ok := s.pool[node]
	if !ok {
		log.Warn("user_key: \"%s\" hit redis node: \"%s\" not in pool", key, node)
		return nil
	}
	log.Debug("user_key: \"%s\" hit redis node: \"%s\"", key, node)
	return p.Get()
}
