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
	Addr             string            `goconf:"base:addr"`
	PKey             string            `goconf:"base:pkey"`
	LogPath          string            `goconf:"log:path"`
	LogLevel         string            `goconf:"log:level"`
	RedisIdleTimeout time.Duration     `goconf:"redis:idletimeout:time"`
	RedisMaxIdle     int               `goconf:"redis:maxidle"`
	RedisMaxActive   int               `goconf:"redis:maxactive"`
	RedisAddrs       map[string]string `goconf:"-"`
}

// Initialize config
func NewConfig(fileName string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(fileName); err != nil {
		return nil, err
	}

	conf := &Config{
		Addr:             ":8070",
		PKey:             "gopushpkey",
		LogPath:          "./message.log",
		LogLevel:         "DEBUG",
		RedisIdleTimeout: 28800 * time.Second,
		RedisMaxIdle:     50,
		RedisMaxActive:   1000,
		RedisAddrs:       make(map[string]string),
	}
	if err := gconf.Unmarshal(conf); err != nil {
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
