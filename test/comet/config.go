package main

import (
	"errors"
	"flag"
	"github.com/Terry-Mao/goconf"
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
	LogFile   string `goconf:"base:logfile"`
	LogLevel  string `goconf:"base:loglevel"`
	Addr      string `goconf:"base:addr"`
	Key       string `goconf:"base:key"`
	Heartbeat int    `goconf:"base:heartbeat"`
}

// InitConfig get a new Config struct.
func InitConfig(file string) (*Config, error) {
	cf := &Config{
		// base
		LogFile:   "./comet-test.log",
		LogLevel:  "ERROR",
		Addr:      "localhost:6969",
		Key:       "Terry-Mao",
		Heartbeat: 30,
	}
	c := goconf.New()
	if err := c.Parse(file); err != nil {
		Log.Error("goconf.Parse(\"%s\") failed (%s)", file, err.Error())
		return nil, err
	}
	if err := c.Unmarshal(cf); err != nil {
		Log.Error("goconf.Unmarshal() failed (%s)", err.Error())
		return nil, err
	}
	return cf, nil
}
