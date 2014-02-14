package main

import (
	"flag"
	"fmt"
	"github.com/Terry-Mao/goconf"
	"runtime"
	"time"
)

var (
	Conf     *Config
	ConfFile string
)

func init() {
	flag.StringVar(&ConfFile, "c", "./message.conf", " set message config file path")
}

// Config struct
type Config struct {
	Addr             string            `goconf:"base:addr"`
	PKey             string            `goconf:"base:pkey"`
	User             string            `goconf:"base:user"`
	PidFile          string            `goconf:"base:pidfile"`
	Dir              string            `goconf:"base:dir"`
	MaxProc          int               `goconf:"base:maxproc"`
	LogFile          string            `goconf:"base:logfile"`
	LogLevel         string            `goconf:"base:loglevel"`
	RedisIdleTimeout time.Duration     `goconf:"redis:idletimeout:time"`
	RedisMaxIdle     int               `goconf:"redis:maxidle"`
	RedisMaxActive   int               `goconf:"redis:maxactive"`
	RedisAddrs       map[string]string `goconf:"-"`
}

// Initialize config
func NewConfig(fileName string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(fileName); err != nil {
		Log.Error("goconf.Parse(\"%s\") error(%v)", fileName, err)
		return nil, err
	}

	conf := &Config{
		Addr:             ":8070",
		PKey:             "gopushpkey",
		User:             "nobody nobody",
		PidFile:          "/tmp/gopush-cluster-message.pid",
		Dir:              "./",
		MaxProc:          runtime.NumCPU(),
		LogFile:          "./message.log",
		LogLevel:         "DEBUG",
		RedisIdleTimeout: 28800 * time.Second,
		RedisMaxIdle:     50,
		RedisMaxActive:   1000,
		RedisAddrs:       make(map[string]string),
	}
	if err := gconf.Unmarshal(conf); err != nil {
		Log.Error("goconf.Unmarshal() error(%v)", err)
		return nil, err
	}

	redisAddrsSec := gconf.Get("redis.addr")
	if redisAddrsSec != nil {
		for _, key := range redisAddrsSec.Keys() {
			addr, err := redisAddrsSec.String(key)
			if err != nil {
				return nil, fmt.Errorf("config section:\"redis.addrs\" key:\"%s\" error(%v)", key, err)
			}
			conf.RedisAddrs[key] = addr
		}
	}

	return conf, nil
}
