package main

import (
	"flag"
	"fmt"
	"github.com/Terry-Mao/goconf"
	"time"
)

var (
	Conf     *Config
	ConfFile string
)

// InitConfig initialize config file path
func InitConfig() {
	flag.StringVar(&ConfFile, "c", "./push.conf", " set push config file path")
}

/*
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
	AdminAddr string           `json:"admin_addr"`
	LogPath   string           `json:"log_path"`
	LogLevel  int              `json:"log_level"`
	Zookeeper *ZookeeperConfig `json:"zookeeper"`
	Push      *PushConfig      `json:"push"`
	MsgSvr    *MessageServer   `json:"msg_svr"`
}*/

type Config struct {
	Addr         string
	AdminAddr    string
	LogPath      string
	LogLevel     int
	ZKAddr       string
	ZKTimeout    time.Duration
	ZKRootPath   string
	CometNetwork string
	MsgAddr      string
	MsgNetwork   string
	MsgHeartbeat time.Duration
	MsgRetry     time.Duration
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
		LogLevel:     0,
		ZKAddr:       ":2181",
		ZKTimeout:    8 * time.Hour,
		ZKRootPath:   "/gopush-cluster",
		CometNetwork: "tcp",
		MsgAddr:      ":8070",
		MsgNetwork:   "tcp",
		MsgHeartbeat: 1 * time.Second,
		MsgRetry:     3 * time.Second,
	}

	bSec := gconf.Get("base")
	if bSec != nil {
		addr, err := bSec.String("addr")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"base\" key:\"addr\" error(%v)", err)
			}
		} else {
			conf.Addr = addr
		}

		adminaddr, err := bSec.String("adminaddr")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"base\" key:\"adminaddr\" error(%v)", err)
			}
		} else {
			conf.AdminAddr = adminaddr
		}
	}

	logSec := gconf.Get("log")
	if logSec != nil {
		logPath, err := logSec.String("path")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"log\" key:\"path\" error(%v)", err)
			}
		} else {
			conf.LogPath = logPath
		}

		level, err := logSec.Int("level")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"log\" key:\"level\" error(%v)", err)
			}
		} else {
			conf.LogLevel = int(level)
		}
	}

	zkSec := gconf.Get("zookeeper")
	if zkSec != nil {
		addr, err := zkSec.String("addr")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"zookeeper\" key:\"addr\" error(%v)", err)
			}
		} else {
			conf.ZKAddr = addr
		}

		timeout, err := zkSec.Duration("timeout")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"zookeeper\" key:\"timeout\" error(%v)", err)
			}
		} else {
			conf.ZKTimeout = time.Duration(timeout)
		}

		rootPath, err := zkSec.String("rootpath")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"zookeeper\" key:\"rootPath\" error(%v)", err)
			}
		} else {
			conf.ZKRootPath = rootPath
		}
	}

	cometSec := gconf.Get("comet")
	if cometSec != nil {
		network, err := cometSec.String("network")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"comet\" key:\"network\" error(%v)", err)
			}
		} else {
			conf.CometNetwork = network
		}
	}

	msgSec := gconf.Get("msg")
	if msgSec != nil {
		addr, err := msgSec.String("addr")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"msg\" key:\"addr\" error(%v)", err)
			}
		} else {
			conf.MsgAddr = addr
		}

		network, err := msgSec.String("network")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"msg\" key:\"network\" error(%v)", err)
			}
		} else {
			conf.CometNetwork = network
		}

		heartbeat, err := msgSec.Duration("heartbeat")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"msg\" key:\"heartbeat\" error(%v)", err)
			}
		} else {
			conf.MsgHeartbeat = time.Duration(heartbeat)
		}

		retry, err := msgSec.Int("retry")
		if err != nil {
			if err != goconf.ErrNoKey {
				return nil, fmt.Errorf("config section:\"msg\" key:\"retry\" error(%v)", err)
			}
		} else {
			conf.MsgRetry = time.Duration(retry)
		}
	}

	return conf, nil
}
