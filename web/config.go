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

func initConfig() {
	flag.StringVar(&ConfFile, "c", "./web.conf", " set web config file path")
	flag.Parse()
}

type Config struct {
	Addr         string
	InternalAddr string
	LogPath      string
	Bucket       uint
	Zookeeper    struct {
		Addr     string
		Timeout  int
		RootPath string
	}
	Redis struct {
		Addr        string
		IdleTimeout int
		MaxIdle     int
	}
}

func NewConfig(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	cf := &Config{
		Addr:         "127.0.0.1:8080",
		InternalAddr: "127.0.0.1:8081",
		LogPath:      "./web.log",
		Bucket:       16,
	}
	cf.Zookeeper.Addr = "10.20.216.122:2181"
	cf.Zookeeper.Timeout = 28800
	cf.Zookeeper.RootPath = "/gopush-cluster"
	cf.Redis.Addr = "10.20.216.122:6379"
	cf.Redis.IdleTimeout = 28800
	cf.Redis.MaxIdle = 50

	if err := json.Unmarshal(c, cf); err != nil {
		return nil, err
	}

	return cf, nil
}
