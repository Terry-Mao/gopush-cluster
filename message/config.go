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
	PprofBind        []string          `goconf:"base:pprof.bind:,"`
	StorageType      string            `goconf:"storage:type"`
	RedisIdleTimeout time.Duration     `goconf:"redis:idletimeout:time"`
	RedisMaxIdle     int               `goconf:"redis:maxidle"`
	RedisMaxActive   int               `goconf:"redis:maxactive"`
	RedisMaxStore    int               `goconf:"redis:maxstore"`
	MYSQLDelLoopTime time.Duration     `goconf:"mysql:dellooptime:time"`
	RedisAddrs       map[string]string `goconf:"-"`
	DBSource         map[string]string `goconf:"-"`
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
		PprofBind:        []string{"localhost:8170"},
		StorageType:      "redis",
		RedisIdleTimeout: 28800 * time.Second,
		RedisMaxIdle:     50,
		RedisMaxActive:   1000,
		RedisMaxStore:    20,
		RedisAddrs:       make(map[string]string),
		MYSQLDelLoopTime: 1 * time.Hour,
		DBSource:         make(map[string]string),
	}
	if err := gconf.Unmarshal(conf); err != nil {
		Log.Error("goconf.Unmarshal() error(%v)", err)
		return nil, err
	}

	//Load redis addresses
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

	//Load mysql sources
	dbSource := gconf.Get("mysql.source")
	if dbSource != nil {
		for _, key := range dbSource.Keys() {
			source, err := dbSource.String(key)
			if err != nil {
				return nil, fmt.Errorf("config section:\"mysql.source\" key:\"%s\" error(%v)", key, err)
			}
			conf.DBSource[key] = source
		}
	}

	return conf, nil
}
