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
	"github.com/golang/glog"
	"runtime"
	"time"
)

var (
	Conf     *Config
	confFile string
)

func init() {
	flag.StringVar(&confFile, "c", "./message.conf", " set message config file path")
}

// Config struct
type Config struct {
	RPCBind          []string          `goconf:"base:rpc.bind:,"`
	User             string            `goconf:"base:user"`
	PidFile          string            `goconf:"base:pidfile"`
	Dir              string            `goconf:"base:dir"`
	MaxProc          int               `goconf:"base:maxproc"`
	PprofBind        []string          `goconf:"base:pprof.bind:,"`
	StorageType      string            `goconf:"storage:type"`
	RedisIdleTimeout time.Duration     `goconf:"redis:timeout:time"`
	RedisMaxIdle     int               `goconf:"redis:idle"`
	RedisMaxActive   int               `goconf:"redis:active"`
	RedisMaxStore    int               `goconf:"redis:store"`
	RedisKetamaBase  int               `goconf:"redis:ketama.base"`
	MySQLClean       time.Duration     `goconf:"mysql:clean:time"`
	MySQLKetamaBase  int               `goconf:"mysql:ketama.base"`
	RedisSource      map[string]string `goconf:"-"`
	MySQLSource      map[string]string `goconf:"-"`
	// zookeeper
	ZookeeperAddr    []string      `goconf:"zookeeper:addr:,"`
	ZookeeperTimeout time.Duration `goconf:"zookeeper:timeout:time"`
	ZookeeperPath    string        `goconf:"zookeeper:path"`
}

// NewConfig parse config file into Config.
func InitConfig() error {
	gconf := goconf.New()
	if err := gconf.Parse(confFile); err != nil {
		glog.Errorf("goconf.Parse(\"%s\") error(%v)", confFile, err)
		return err
	}
	Conf = &Config{
		// base
		RPCBind:   []string{"localhost:8070"},
		User:      "nobody nobody",
		PidFile:   "/tmp/gopush-cluster-message.pid",
		Dir:       "./",
		MaxProc:   runtime.NumCPU(),
		PprofBind: []string{"localhost:8170"},
		// storage
		StorageType: "redis",
		// redis
		RedisIdleTimeout: 28800 * time.Second,
		RedisMaxIdle:     50,
		RedisMaxActive:   1000,
		RedisMaxStore:    20,
		RedisSource:      make(map[string]string),
		// mysql
		MySQLSource: make(map[string]string),
		MySQLClean:  1 * time.Hour,
		// zookeeper
		ZookeeperAddr:    []string{"localhost:2181"},
		ZookeeperTimeout: 30 * time.Second,
		ZookeeperPath:    "/gopush-cluster-message",
	}
	if err := gconf.Unmarshal(Conf); err != nil {
		glog.Errorf("goconf.Unmarshal() error(%v)", err)
		return err
	}
	// redis section
	redisAddrsSec := gconf.Get("redis.source")
	if redisAddrsSec != nil {
		for _, key := range redisAddrsSec.Keys() {
			addr, err := redisAddrsSec.String(key)
			if err != nil {
				return fmt.Errorf("config section: \"redis.addrs\" key: \"%s\" error(%v)", key, err)
			}
			Conf.RedisSource[key] = addr
		}
	}
	// mysql section
	dbSource := gconf.Get("mysql.source")
	if dbSource != nil {
		for _, key := range dbSource.Keys() {
			source, err := dbSource.String(key)
			if err != nil {
				return fmt.Errorf("config section: \"mysql.source\" key: \"%s\" error(%v)", key, err)
			}
			Conf.MySQLSource[key] = source
		}
	}
	return nil
}
