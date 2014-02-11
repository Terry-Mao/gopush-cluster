package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"github.com/garyburd/redigo/redis"
)

const (
	defaultRedisNode = "node1"
)

var (
	RedisNoConnErr = errors.New("can't get a redis conn")
	redisPool      = map[string]*redis.Pool{}
	redisHash      *hash.Ketama
)

// Struct for delele message
type DelMessageInfo struct {
	Key  string
	Msgs []string
}

// Initialize redis pool, Initialize consistency hash ring
func InitRedis() {
	// Redis pool
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
					Log.Error("redis.Dial(\"tcp\", \"%s\") error(%v)", tmp, err)
				}
				return conn, err
			},
		}
	}

	// Consistent hashing
	redisHash = hash.NewKetama(len(redisPool), 255)
}

// SaveMessage save offline messages
func SaveMessage(key, msg string, mid int64) error {
	conn := getRedisConn(key)
	if conn == nil {
		return RedisNoConnErr
	}

	defer conn.Close()
	_, err := redis.Int(conn.Do("ZADD", key, mid, msg))
	if err != nil {
		return err
	}

	return nil
}

// GetMessages get all of offline messages which larger than mid
func GetMessages(key string, mid int64) ([]string, error) {
	conn := getRedisConn(key)
	if conn == nil {
		return nil, RedisNoConnErr
	}

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

// Delete Message
func DelMessages(info *DelMessageInfo) error {
	commands := []struct {
		args []interface{}
	}{}

	for i := 0; i < len(info.Msgs); i++ {
		commands = append(commands,
			struct {
				args []interface{}
			}{
				args: []interface{}{"ZREM", info.Key, info.Msgs[i]},
			},
		)
	}

	conn := getRedisConn(info.Key)
	if conn == nil {
		return RedisNoConnErr
	}
	defer conn.Close()

	for _, cmd := range commands {
		if err := conn.Send(cmd.args[0].(string), cmd.args[1:]...); err != nil {
			return err
		}
	}

	if err := conn.Flush(); err != nil {
		return err
	}

	for _, _ = range commands {
		_, err := conn.Receive()
		if err != nil {
			return err
		}
	}

	return nil
}

// getRedisConn get the redis connection of matching with key
func getRedisConn(key string) redis.Conn {
	node := defaultRedisNode
	// if multiple redispool use ketama
	if len(redisPool) != 1 {
		node = redisHash.Node(key)
	}

	p, ok := redisPool[node]
	if !ok {
		Log.Warn("no exists key:\"%s\" in redisPool map", key)
		return nil
	}

	Log.Debug("key:\"%s\", hit node:\"%s\"", key, node)
	return p.Get()
}
