package main

import (
	"errors"
	"flag"
	"github.com/Terry-Mao/goconf"
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
	User          string   `goconf:"base:user"`
	PidFile       string   `goconf:"base:pidfile"`
	Dir           string   `goconf:"base:dir"`
	MaxProc       int      `goconf:"base:maxproc"`
	LogFile       string   `goconf:"base:logfile"`
	LogLevel      string   `goconf:"base:loglevel"`
	TCPBind       []string `goconf:"base:tcp.bind:,"`
	WebsocketBind []string `goconf:"base:websocket.bind:,"`
	RPCBind       []string `goconf:"base:rpc.bind:,"`
	PprofBind     []string `goconf:"base:pprof.bind:,"`
	// zookeeper
	ZookeeperAddr    string        `goconf:"zookeeper:addr"`
	ZookeeperTimeout time.Duration `goconf:"zookeeper:timeout:time"`
	ZookeeperPath    string        `goconf:"zookeeper:path"`
	ZookeeperNode    string        `goconf:"zookeeper:node"`
	// rpc
	RPCMessageAddr string        `goconf:"rpc:message.addr"`
	RPCPing        time.Duration `goconf:"rpc:ping:time"`
	RPCRetry       time.Duration `goconf:"rpc:retry:time"`
	// channel
	SndbufSize              int           `goconf:"channel:sndbuf.size:memory"`
	RcvbufSize              int           `goconf:"channel:rcvbuf.size:memory"`
	Proto                   []string      `goconf:"channel:proto:,"`
	BufioInstance           int           `goconf:"channel:bufio.instance"`
	BufioNum                int           `goconf:"channel:bufio.num"`
	TCPKeepalive            bool          `goconf:"channel:tcp.keepalive"`
	MaxSubscriberPerChannel int           `goconf:"channel:maxsubscriber"`
	ChannelBucket           int           `goconf:"channel:bucket"`
	Auth                    bool          `goconf:"channel:auth"`
	TokenExpire             time.Duration `goconf:"-"`
}

// InitConfig get a new Config struct.
func InitConfig(file string) (*Config, error) {
	cf := &Config{
		// base
		User:          "nobody nobody",
		PidFile:       "/tmp/gopush-cluster-comet.pid",
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
		Proto:                   []string{"tcp", "websocket"},
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
		Log.Error("goconf.Parse(\"%s\") error(%v)", file, err)
		return nil, err
	}
	if err := c.Unmarshal(cf); err != nil {
		Log.Error("gocong.Unmarshall() error(%v)", err)
	}
	return cf, nil
}
