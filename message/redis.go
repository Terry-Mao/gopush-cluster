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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
)

const (
	defaultRedisNode = "node1"
)

var (
	RedisNoConnErr = errors.New("can't get a redis conn")
)

// RedisMessage struct encoding the composite info.
type RedisMessage struct {
	Msg json.RawMessage `json:"msg"`// message content 
    GroupId int `json:"gid"`// group id
    Expire int64 `json:"expire"`// expire second
}

type RedisStorage struct {
	Pool   map[string]*redis.Pool
	Ketama *hash.Ketama
}

// NewRedis initialize the redis pool and consistency hash ring.
func NewRedis() *RedisStorage {
	redisPool := map[string]*redis.Pool{}
	for n, addr := range Conf.RedisAddrs {
		tmp := addr
		// WARN: closures use
		redisPool[n] = &redis.Pool{
			MaxIdle:     Conf.RedisMaxIdle,
			MaxActive:   Conf.RedisMaxActive,
			IdleTimeout: Conf.RedisIdleTimeout,
			Dial: func() (redis.Conn, error) {
				conn, err := redis.Dial("tcp", tmp)
				if err != nil {
					glog.Errorf("redis.Dial(\"tcp\", \"%s\") error(%v)", tmp, err)
				}
				return conn, err
			},
		}
	}
	return &RedisStorage{Pool: redisPool, Ketama: hash.NewKetama(len(redisPool), 255)}
}

// Save implements the Storage Save method.
func (s *RedisStorage) Save(key string, msg json.RawMessage, mid int64, gid int, expire uint) error {
	conn := s.getConn(key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()
    rm := &RedisMessage{Msg: msg, GroupId: gid, Expire: expire}
    m, err := json.Marshal(rm)
    if err != nil {
        glog.Errorf("json.Marshal(\"%v\") error(%v)", rm, err)
        return err
    }
	//ZADD
	if err := conn.Send("ZADD", key, mid, m); err != nil {
        glog.Errorf("conn.Send(\"ZADD\", \"%s\", %d, \"%s\") error(%v)", key, mid, string(m), err)
		return err
	}
	//ZREMRANGEBYRANK
	if err := conn.Send("ZREMRANGEBYRANK", key, 0, -1*(Conf.RedisMaxStore+1)); err != nil {
        glog.Errorf("conn.Send(\"ZREMRANGEBYRANK\", \"%s\", 0, %d) error(%v)", key, mid, -1*(Conf.RedisMaxStore+1), err)
		return err
	}
	if err := conn.Flush(); err != nil {
        glog.Errorf("conn.Flush() error(%v)", err)
		return err
	}
	//ZADD Receive
	_, err := conn.Receive()
	if err != nil {
        glog.Errorf("conn.Receive() error(%v)", err)
		return err
	}
	//ZREMRANGEBYRANK Receive
	_, err = conn.Receive()
	if err != nil {
        glog.Errorf("conn.Receive() error(%v)", err)
		return err
	}
	return nil
}

// Save implements the Storage Get method.
func (s *RedisStorage) Get(key string, mid int64) ([]interface{}, error) {
    var (
        cmid int64
        b    []byte
        now = time.Now().Unix()
    )
	conn := s.getConn(key)
	if conn == nil {
		return nil, RedisNoConnErr
	}
	defer conn.Close()
	values, err := redis.Values(conn.Do("ZRANGEBYSCORE WITHSCORES", key, fmt.Sprintf("(%d", mid), "+inf"))
	if err != nil {
        glog.Errorf("conn.Do(\"ZRANGEBYSCORE\", \"%s\", \"%s\", \"+inf\") error(%v)", err)
		return nil, err
	}
    msgs := []*Message{}
	for len(values) > 0 {
        rm := &RedisMessage{}
		values, err = redis.Scan(values, &cmid, &b)
		if err != nil {
			logger.Error("redis.Scan() error(%v)", err)
            return nil, err
		}
        if err := json.Unmarshal(b, rm); err != nil {
            glog.Errorf("json.Unmarshal(\"%s\", rm) error(%v)", string(b), err)
            // TODO del
        }
        // check expire
        if rm.Expire < now {
            glog.Warningf("user_key: \"%s\" msg: %d expired", key, cmid)
            continue
        }
        m := &Message{MsgId: cmid, Msg: rm.Msg, GroupId: rm.GroupId}
		msgs = append(msgs, m)
	}
    return msgs, nil
}

// DelMulti implements the Storage DelMulti method.
func (s *RedisStorage) DelMulti(info *DelMessageInfo) error {
	conn := s.getConn(info.Key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()

	for _, msg := range info.Msgs {
		if err := conn.Send("ZREM", info.Key, msg); err != nil {
			return err
		}
	}

	if err := conn.Flush(); err != nil {
		return err
	}

	for _, _ = range info.Msgs {
		_, err := conn.Receive()
		if err != nil {
			return err
		}
	}

	return nil
}

// DelKey implements the Storage DelKey method.
func (s *RedisStorage) DelKey(key string) error {
	conn := s.getConn(key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}

	return nil
}

// getConn get the connection of matching with key using ketama hashing.
func (s *RedisStorage) getConn(key string) redis.Conn {
	node := defaultRedisNode
	if len(s.Pool) > 1 {
		node = s.Ketama.Node(key)
	}

	p, ok := s.Pool[node]
	if !ok {
		glog.Warningf("no exists key:\"%s\" in redis pool", key)
		return nil
	}

	glog.V(1).Infof("key:\"%s\", hit node:\"%s\"", key, node)
	return p.Get()
}
