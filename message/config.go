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
	flag.StringVar(&ConfFile, "c", "./message.conf", " set message config file path")
}

type RedisConfig struct {
	Network     string `json:"network"`
	Addr        string `json:"addr"`
	IdleTimeout int    `json:"idle_timeout"`
	MaxIdle     int    `json:"max_idle"`
	MaxActive   int    `json:"max_active"`
}

type Config struct {
	Addr     string                  `json:"addr"`
	LogPath  string                  `json:"log_path"`
	LogLevel int                     `json:"log_level"`
	Redis    map[string]*RedisConfig `json:"redis"`
}

// NewConfig get a config
func NewConfig(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read config file fail (%v)", err))
	}

	// Default config
	cf := &Config{
		Addr:     "127.0.0.1:8082",
		LogPath:  "./message.log",
		LogLevel: 0,
	}

	// Parse config file
	if err := json.Unmarshal(c, cf); err != nil {
		return nil, errors.New(fmt.Sprintf("parse config file fail (%v)", err))
	}

	return cf, nil
}
