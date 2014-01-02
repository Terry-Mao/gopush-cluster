package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
)

var (
	Conf     *Config
	ConfFile string
)

// InitConfig initialize config file path
func InitConfig() {
	flag.StringVar(&ConfFile, "c", "./push.conf", " set push config file path")
}

type ZookeeperConfig struct {
	Addr     string `json:"addr"`
	Timeout  int    `json:"timeout"` //second
	RootPath string `json:"root_path"`
}

type PushConfig struct {
	DialTimeout int    `json:"dial_timeout"` //second
	Deadline    int    `json:"deadline"`     //second
	Network     string `json:"network"`
}

type MessageServer struct {
	Addr      string `json:"addr"`
	Network   string `json:"network"`
	Heartbeat int    `json:"heartbeat"` //second
	Retry     int    `json:"retry"`     //second
}

type Config struct {
	Addr      string           `json:"addr"`
	LogPath   string           `json:"log_path"`
	LogLevel  int              `json:"log_level"`
	Bucket    uint             `json:"bucket"`
	Zookeeper *ZookeeperConfig `json:"zookeeper"`
	Push      *PushConfig      `json:"push"`
	MsgSvr    *MessageServer   `json:"msg_svr"`
}

// NewConfig get a config
func NewConfig(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read config file fail (%v)", err))
	}

	// Default config
	cf := &Config{
		Addr:     "127.0.0.1:8080",
		LogPath:  "./web.log",
		LogLevel: 0,
		Bucket:   16,
	}

	// Parse config file
	if err := json.Unmarshal(c, cf); err != nil {
		return nil, errors.New(fmt.Sprintf("parse config file fail (%v)", err))
	}

	return cf, nil
}
