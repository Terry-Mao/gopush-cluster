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
	"errors"
	"flag"
	"github.com/Terry-Mao/goconf"
	"github.com/golang/glog"
)

var (
	Conf               *Config
	ConfFile           string
	ErrNoConfigSection = errors.New("no config section")
)

func init() {
	flag.StringVar(&ConfFile, "c", "./comet-test.conf", " set gopush-cluster comet test config file path")
}

type Config struct {
	// base
	Addr      string `goconf:"base:addr"`
	Key       string `goconf:"base:key"`
	Heartbeat int    `goconf:"base:heartbeat"`
}

// InitConfig get a new Config struct.
func InitConfig(file string) (*Config, error) {
	cf := &Config{
		// base
		Addr:      "localhost:6969",
		Key:       "Terry-Mao",
		Heartbeat: 30,
	}
	c := goconf.New()
	if err := c.Parse(file); err != nil {
		glog.Errorf("goconf.Parse(\"%s\") failed (%s)", file, err.Error())
		return nil, err
	}
	if err := c.Unmarshal(cf); err != nil {
		glog.Errorf("goconf.Unmarshal() failed (%s)", err.Error())
		return nil, err
	}
	return cf, nil
}
