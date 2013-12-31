package main

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

var (
	redisPool *redis.Pool
)

func initRedis() {
	redisPool = &redis.Pool{
		MaxIdle:     Conf.Redis.MaxIdle,
		IdleTimeout: time.Duration(Conf.Redis.IdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", Conf.Redis.Addr)
			if err != nil {
				panic(err)
			}

			return c, err
		},
	}
}

func SaveMessage(key, msg string, mid int64) error {
	conn := redisPool.Get()
	defer conn.Close()

	reply, err := redis.Int(conn.Do("ZADD", key, mid, msg))
	if err != nil {
		return err
	}

	if reply != 1 {
		return errors.New(fmt.Sprintf("ZADD fail, reply:%d", reply))
	}

	return nil
}

func GetMessages(key string, mid int64) ([]string, error) {
	conn := redisPool.Get()
	defer conn.Close()

	reply, err := redis.Strings(conn.Do("ZRANGEBYSCORE", key, fmt.Sprintf("(%d", mid), "+inf"))
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}

		return nil, err
	}

	return reply, nil
}
