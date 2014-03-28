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
	Addr        string        `goconf:"base:addr"`
	AdminAddr   string        `goconf:"base:adminaddr"`
	MaxProc     int           `goconf:"base:maxproc"`
	PprofBind   []string      `goconf:"base:pprof.bind:,"`
	User        string        `goconf:"base:user"`
	PidFile     string        `goconf:"base:pidfile"`
	Dir         string        `goconf:"base:dir"`
	LogPath     string        `goconf:"log:path"`
	LogLevel    string        `goconf:"log:level"`
	ZKAddr      []string      `goconf:"zookeeper:addr:,"`
	ZKTimeout   time.Duration `goconf:"zookeeper:timeout:time"`
	ZKCometPath string        `goconf:"zookeeper:cometpath"`
	ZKPIDPath   string        `goconf:"zookeeper:pidpath"`
	MsgAddr     string        `goconf:"msg:addr"`
	MsgPing     time.Duration `goconf:"msg:ping:time"`
	MsgRetry    time.Duration `goconf:"msg:retry:time"`
}

// Initialize config
func NewConfig(file string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		return nil, err
	}

	// Default config
	conf := &Config{
		Addr:        ":80",
		AdminAddr:   ":81",
		MaxProc:     runtime.NumCPU(),
		PprofBind:   []string{"localhost:8190"},
		User:        "nobody nobody",
		PidFile:     "/tmp/gopush-cluster-web.pid",
		Dir:         "./",
		LogPath:     "./web.log",
		LogLevel:    "DEBUG",
		ZKAddr:      []string{":2181"},
		ZKTimeout:   30 * time.Second,
		ZKCometPath: "/gopush-cluster",
		ZKPIDPath:   "/gopush-pid",
		MsgAddr:     ":8070",
		MsgPing:     1 * time.Second,
		MsgRetry:    3 * time.Second,
	}

	if err := gconf.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
