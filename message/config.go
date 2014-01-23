package main

import (
	"flag"
	"fmt"
	"github.com/Terry-Mao/goconf"
	"time"
)

var (
	Conf     *Config
	ConfFile string
)

// InitConfig initialize config file path
func InitConfig() {
	flag.StringVar(&ConfFile, "c", "./message.conf", " set message config file path")
}

// Config struct
type Config struct {
	Addr             string
	LogPath          string
	LogLevel         int
	RedisNetwork     string
	RedisIdleTimeout time.Duration
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisAddrs       map[string]string
}

// Initialize config
func NewConfig(fileName string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(fileName); err != nil {
		return nil, err
	}

	conf := &Config{
		Addr:             ":8070",
		LogPath:          "./message.log",
		LogLevel:         7,
		RedisNetwork:     "tcp",
		RedisIdleTimeout: 28800 * time.Second,
		RedisMaxIdle:     50,
		RedisMaxActive:   1000,
		RedisAddrs:       make(map[string]string),
	}

	bSec := gconf.Get("base")
	if bSec != nil {
		addr, err := bSec.String("addr")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"base\" key:\"addr\" error(%v)", err)
			}
		} else {
			conf.Addr = addr
		}
	}

	logSec := gconf.Get("log")
	if logSec != nil {
		logPath, err := logSec.String("path")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"log\" key:\"path\" error(%v)", err)
			}
		} else {
			conf.LogPath = logPath
		}

		level, err := logSec.Int("level")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"log\" key:\"level\" error(%v)", err)
			}
		} else {
			conf.LogLevel = int(level)
		}
	}

	redisSec := gconf.Get("redis")
	if redisSec != nil {
		redisNetwork, err := redisSec.String("network")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"redis\" key:\"network\" error(%v)", err)
			}
		} else {
			conf.RedisNetwork = redisNetwork
		}

		redisIdleTimeout, err := redisSec.Duration("idletimeout")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"redis\" key:\"idletimeout\" error(%v)", err)
			}
		} else {
			conf.RedisIdleTimeout = time.Duration(redisIdleTimeout)
		}

		maxidle, err := redisSec.Int("maxidle")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"redis\" key:\"maxidle\" error(%v)", err)
			}
		} else {
			conf.RedisMaxIdle = int(maxidle)
		}

		maxactive, err := redisSec.Int("maxactive")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"redis\" key:\"maxactive\" error(%v)", err)
			}
		} else {
			conf.RedisMaxActive = int(maxactive)
		}
	}

	redisAddrsSec := gconf.Get("redis.addr")
	if redisAddrsSec != nil {
		for _, key := range redisAddrsSec.Keys() {
			addr, err := redisAddrsSec.String("key")
			if err != nil {
				return nil, fmt.Errorf("config section:\"redis.addrs\" key:\"%s\" error(%v)", key, err)
			}
			conf.RedisAddrs[key] = addr
		}
	}

	return conf, nil
}
