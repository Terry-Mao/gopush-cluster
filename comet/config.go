package main

import (
	"errors"
	"flag"
	"github.com/Terry-Mao/goconf"
	"github.com/Terry-Mao/gopush-cluster/log"
	"runtime"
	"time"
)

var (
	Conf               *Config
	ConfFile           string
	ErrNoConfigSection = errors.New("no config section")
)

func init() {
	flag.StringVar(&ConfFile, "c", "./comet.conf", " set gopush-cluster comet config file path")
}

type Config struct {
	// base
	User          string
	PidFile       string
	Dir           string
	MaxProc       int
	LogFile       string
	LogLevel      string
	TCPBind       []string
	WebsocketBind []string
	RPCBind       []string
	PprofBind     []string
	// zookeeper
	ZookeeperAddr    string
	ZookeeperTimeout time.Duration
	ZookeeperPath    string
	ZookeeperNode    string
	// rpc
	RPCMessageAddr string
	RPCPing        time.Duration
	RPCRetry       time.Duration
	// channel
	SndbufSize              int
	RcvbufSize              int
	Proto                   string
	BufioInstance           int
	BufioNum                int
	TCPKeepalive            bool
	MaxSubscriberPerChannel int
	ChannelBucket           int
	Auth                    bool
	TokenExpire             time.Duration
}

// InitConfig get a new Config struct.
func InitConfig(file string) (*Config, error) {
	cf := &Config{
		// base
		User:          "nobody nobody",
		PidFile:       "/var/run/gopush-cluster-comet.pid",
		Dir:           "./",
		MaxProc:       runtime.NumCPU(),
		LogFile:       "./comet.log",
		LogLevel:      "ERROR",
		WebsocketBind: []string{"localhost:6968"},
		TCPBind:       []string{"localhost:6969"},
		RPCBind:       []string{"localhost:6970"},
		PprofBind:     []string{"localhost:6971"},
		// zookeeper
		ZookeeperAddr:    "localhost:2181",
		ZookeeperTimeout: 8 * time.Hour,
		ZookeeperPath:    "/gopush-cluster",
		ZookeeperNode:    "node1",
		// rpc
		RPCMessageAddr: "localhost:6972",
		RPCPing:        1 * time.Second,
		RPCRetry:       1 * time.Second,
		// channel
		SndbufSize:              2048,
		RcvbufSize:              256,
		Proto:                   "tcp",
		BufioInstance:           runtime.NumCPU(),
		BufioNum:                128,
		TCPKeepalive:            false,
		TokenExpire:             30 * 24 * time.Hour,
		MaxSubscriberPerChannel: 64,
		ChannelBucket:           runtime.NumCPU(),
		Auth:                    false,
	}
	c := goconf.New()
	if err := c.Parse(file); err != nil {
		log.DefaultLogger.Error("goconf.Parse(\"%s\") failed (%s)", file, err.Error())
		return nil, err
	}
	// base section
	baseSection := c.Get("base")
	if baseSection == nil {
		return nil, ErrNoConfigSection
	}
	if err := setConfigDefString(baseSection, "user", &cf.User); err != nil {
		return nil, err
	}
	if err := setConfigDefString(baseSection, "pidfile", &cf.PidFile); err != nil {
		return nil, err
	}
	if err := setConfigDefString(baseSection, "dir", &cf.Dir); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(baseSection, "maxproc", &cf.MaxProc); err != nil {
		return nil, err
	}
	if err := setConfigDefString(baseSection, "logfile", &cf.LogFile); err != nil {
		return nil, err
	}
	if err := setConfigDefString(baseSection, "loglevel", &cf.LogLevel); err != nil {
		return nil, err
	}
	if err := setConfigDefStrings(baseSection, "tcp.bind", ",", cf.TCPBind); err != nil {
		return nil, err
	}
	if err := setConfigDefStrings(baseSection, "websocket.bind", ",", cf.WebsocketBind); err != nil {
		return nil, err
	}
	if err := setConfigDefStrings(baseSection, "pprof.bind", ",", cf.PprofBind); err != nil {
		return nil, err
	}
	if err := setConfigDefStrings(baseSection, "rpc.bind", ",", cf.RPCBind); err != nil {
		return nil, err
	}
	// zookeeper section
	zkSection := c.Get("zookeeper")
	if zkSection == nil {
		return nil, ErrNoConfigSection
	}
	if err := setConfigDefString(zkSection, "addr", &cf.ZookeeperAddr); err != nil {
		return nil, err
	}
	if err := setConfigDefDuration(zkSection, "timeout", &cf.ZookeeperTimeout); err != nil {
		return nil, err
	}
	if err := setConfigDefString(zkSection, "path", &cf.ZookeeperPath); err != nil {
		return nil, err
	}
	if err := setConfigDefString(zkSection, "node", &cf.ZookeeperNode); err != nil {
		return nil, err
	}
	// rpc section
	rpcSection := c.Get("rpc")
	if rpcSection == nil {
		return nil, ErrNoConfigSection
	}
	if err := setConfigDefString(rpcSection, "message.addr", &cf.RPCMessageAddr); err != nil {
		return nil, err
	}
	if err := setConfigDefDuration(rpcSection, "ping", &cf.RPCPing); err != nil {
		return nil, err
	}
	if err := setConfigDefDuration(rpcSection, "retry", &cf.RPCRetry); err != nil {
		return nil, err
	}
	// channel section
	chSection := c.Get("channel")
	if chSection == nil {
		return nil, ErrNoConfigSection
	}
	if err := setConfigDefBool(chSection, "tcp.keepalive", &cf.TCPKeepalive); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "sndbuf.size", &cf.SndbufSize); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "rcvbuf.size", &cf.RcvbufSize); err != nil {
		return nil, err
	}
	if err := setConfigDefString(chSection, "proto", &cf.Proto); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "bufio.instance", &cf.BufioInstance); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "bufio.num", &cf.BufioNum); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "maxsubscriber", &cf.MaxSubscriberPerChannel); err != nil {
		return nil, err
	}
	if err := setConfigDefInt(chSection, "bucket", &cf.ChannelBucket); err != nil {
		return nil, err
	}
	if err := setConfigDefBool(chSection, "auth", &cf.Auth); err != nil {
		return nil, err
	}
	if err := setConfigDefDuration(chSection, "token.expire", &cf.TokenExpire); err != nil {
		return nil, err
	}
	return cf, nil
}

func setConfigDefInt(s *goconf.Section, key string, val *int) error {
	if tmp, err := s.Int(key); err != nil {
		if err == goconf.ErrNoKey {
			log.DefaultLogger.Warn("%s directive:\"%s\" not set, use default:\"%d\"", s.Name, key, *val)
		} else {
			log.DefaultLogger.Error("%s.Int(\"%s\") failed (%s)", s.Name, key, err.Error())
			return err
		}
	} else {
		*val = int(tmp)
	}
	return nil
}

func setConfigDefDuration(s *goconf.Section, key string, val *time.Duration) error {
	if tmp, err := s.Duration(key); err != nil {
		if err == goconf.ErrNoKey {
			log.DefaultLogger.Warn("%s directive:\"%s\" not set, use default:\"%d\"", s.Name, key, *val)
		} else {
			log.DefaultLogger.Error("%s.Duration(\"%s\") failed (%s)", s.Name, key, err.Error())
			return err
		}
	} else {
		*val = tmp
	}
	return nil
}

func setConfigDefString(s *goconf.Section, key string, val *string) error {
	if tmp, err := s.String(key); err != nil {
		if err == goconf.ErrNoKey {
			log.DefaultLogger.Warn("%s directive:\"%s\" not set, use default:\"%s\"", s.Name, key, *val)
		} else {
			log.DefaultLogger.Error("%s.String(\"%s\") failed (%s)", s.Name, key, err.Error())
			return err
		}
	} else {
		*val = tmp
	}
	return nil
}

func setConfigDefStrings(s *goconf.Section, key, delim string, val []string) error {
	if tmp, err := s.Strings(key, delim); err != nil {
		if err == goconf.ErrNoKey {
			log.DefaultLogger.Warn("%s directive:\"%s\" not set, use default:\"%v\"", s.Name, key, val)
		} else {
			log.DefaultLogger.Error("%s.Strings(\"%s\") failed (%s)", s.Name, key, err.Error())
			return err
		}
	} else {
		val = tmp
	}
	return nil
}

func setConfigDefBool(s *goconf.Section, key string, val *bool) error {
	if tmp, err := s.Bool(key); err != nil {
		if err == goconf.ErrNoKey {
			log.DefaultLogger.Warn("%s directive:\"%s\" not set, use default:\"%s\"", s.Name, key, *val)
		} else {
			log.DefaultLogger.Error("%s.Bool(\"%s\") failed (%s)", s.Name, key, err.Error())
			return err
		}
	} else {
		*val = tmp
	}
	return nil
}
