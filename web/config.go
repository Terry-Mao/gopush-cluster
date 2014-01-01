package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
)

var (
	Conf     *Config
	ConfFile string
)

// InitConfig initialize config file path
func InitConfig() {
	flag.StringVar(&ConfFile, "c", "./web.conf", " set web config file path")
}

type Config struct {
	Addr         string `json:"addr"`
	InternalAddr string `json:"internal_addr"`
	LogPath      string `json:"log_path"`
	LogLevel     int    `json:"log_level"`
	Bucket       uint   `json:"bucket"`
	Zookeeper    struct {
		Addr     string `json:"addr"`
		Timeout  int    `json:"timeout"`
		RootPath string `json:"rootpath"`
	}
	Redis struct {
		Addr        string `json:"addr"`
		IdleTimeout int    `json:"idle_timeout"`
		MaxIdle     int    `json:"max_idle"`
	}
}

// InitConfig get a config
func NewConfig(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// Default config
	cf := &Config{
		Addr:         "127.0.0.1:8080",
		InternalAddr: "127.0.0.1:8081",
		LogPath:      "./web.log",
		LogLevel:     0,
		Bucket:       16,
	}
	cf.Zookeeper.Addr = "10.20.216.122:2181"
	cf.Zookeeper.Timeout = 28800
	cf.Zookeeper.RootPath = "/gopush-cluster"
	cf.Redis.Addr = "10.20.216.122:6379"
	cf.Redis.IdleTimeout = 28800
	cf.Redis.MaxIdle = 50

	// Parse config file
	if err := json.Unmarshal(c, cf); err != nil {
		return nil, err
	}

	return cf, nil
}
