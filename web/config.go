package main

import (
	"flag"
	"github.com/Terry-Mao/goconf"
	"time"
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
	Addr         string        `goconf:"base:addr"`
	AdminAddr    string        `goconf:"base:adminaddr"`
	LogPath      string        `goconf:"log:path"`
	LogLevel     string        `goconf:"log:level"`
	ZKAddr       string        `goconf:"zookeeper:addr"`
	ZKTimeout    time.Duration `goconf:"zookeeper:timeout:time"`
	ZKRootPath   string        `goconf:"zookeeper:rootpath"`
	CometNetwork string        `goconf:"comet:network"`
	MsgAddr      string        `goconf:"msg:addr"`
	MsgNetwork   string        `goconf:"msg:network"`
	MsgHeartbeat time.Duration `goconf:"msg:heartbeat:time"`
	MsgRetry     time.Duration `goconf:"msg:retry:time"`
}

// Initialize config
func NewConfig(file string) (*Config, error) {
	gconf := goconf.New()
	if err := gconf.Parse(file); err != nil {
		return nil, err
	}

	// Default config
	conf := &Config{
		Addr:         ":80",
		AdminAddr:    ":81",
		LogPath:      "./web.log",
		LogLevel:     "DEBUG",
		ZKAddr:       ":2181",
		ZKTimeout:    8 * time.Hour,
		ZKRootPath:   "/gopush-cluster",
		CometNetwork: "tcp",
		MsgAddr:      ":8070",
		MsgNetwork:   "tcp",
		MsgHeartbeat: 1 * time.Second,
		MsgRetry:     3 * time.Second,
	}

	if err := gconf.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
