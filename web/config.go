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
	"github.com/Terry-Mao/goconf"
	"github.com/golang/glog"
	"runtime"
	"time"
)

var (
	Conf     *Config
	ConfFile string
)

// InitConfig initialize config file path
func init() {
	flag.StringVar(&ConfFile, "c", "./web.conf", " set web config file path")
}

type Config struct {
	Addr                 []string      `goconf:"base:addr:,"`
	AdminAddr            []string      `goconf:"base:admin.addr:,"`
	MaxProc              int           `goconf:"base:maxproc"`
	PprofBind            []string      `goconf:"base:pprof.bind:,"`
	User                 string        `goconf:"base:user"`
	PidFile              string        `goconf:"base:pidfile"`
	Dir                  string        `goconf:"base:dir"`
	Router               string        `goconf:"base:router"`
	QQWryPath            string        `goconf:"res:qqwry.path"`
	ZookeeperAddr        []string      `goconf:"zookeeper:addr:,"`
	ZookeeperTimeout     time.Duration `goconf:"zookeeper:timeout:time"`
	ZookeeperCometPath   string        `goconf:"zookeeper:comet.path"`
	ZookeeperMessagePath string        `goconf:"zookeeper:message.path"`
	RPCRetry             time.Duration `goconf:"rpc:retry:time"`
	RPCPing              time.Duration `goconf:"rpc:ping:time"`
}

// Initialize config
func NewConfig(file string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		glog.Errorf("goconf.Parse(\"%s\") error(%v)", file, err)
		return nil, err
	}
	// Default config
	conf := &Config{
		Addr:                 []string{"localhost:80"},
		AdminAddr:            []string{"localhost:81"},
		MaxProc:              runtime.NumCPU(),
		PprofBind:            []string{"localhost:8190"},
		User:                 "nobody nobody",
		PidFile:              "/tmp/gopush-cluster-web.pid",
		Dir:                  "./",
		Router:               "",
		QQWryPath:            "/tmp/QQWry.dat",
		ZookeeperAddr:        []string{":2181"},
		ZookeeperTimeout:     30 * time.Second,
		ZookeeperCometPath:   "/gopush-cluster-comet",
		ZookeeperMessagePath: "/gopush-cluster-message",
		RPCRetry:             3 * time.Second,
		RPCPing:              1 * time.Second,
	}
	if err := gconf.Unmarshal(conf); err != nil {
		glog.Errorf("goconf.Unmarshall() error(%v)", err)
		return nil, err
	}
	return conf, nil
}
