package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"runtime"
)

var (
	Conf     *Config
	ConfFile string
)

func init() {
	flag.StringVar(&ConfFile, "c", "./gopush2.conf", " set gopush2 config file path")
}

type RedisConfig struct {
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Timeout int    `json:"timeout"`
	Active  int    `json:"active"`
	Idle    int    `json:"idle"`
}

type Config struct {
	Node                string                  `json:"node"`
	Addr                string                  `json:"addr"`
	AdminAddr           string                  `json:"admin_addr"`
	Log                 string                  `json:"log"`
	LogLevel            int                     `json:"log_level"`
	MessageExpireSec    int64                   `json:"message_expire_sec"`
	ChannelExpireSec    int64                   `json:"channel_expire_sec"`
	MaxStoredMessage    int                     `json:"max_stored_message"`
	MaxProcs            int                     `json:"max_procs"`
	MaxSubscriberPerKey int                     `json:"max_subscriber_per_key"`
	TCPKeepAlive        int                     `json:"tcp_keepalive"`
	ChannelBucket       int                     `json:"channel_bucket"`
	ChannelType         int                     `json:"channel_type"`
	HeartbeatSec        int                     `json:"heartbeat_sec"`
	Auth                int                     `json:"auth"`
	Redis               map[string]*RedisConfig `json:"redis"`
	ReadBufInstance     int                     `json:"read_buf_instance"`
	ReadBufNumPerInst   int                     `json:"read_buf_num_per_inst"`
	ReadBufByte         int                     `json:"read_buf_byte"`
	WriteBufNum         int                     `json:"write_buf_num"`
	WriteBufByte        int                     `json:"write_buf_byte"`
	Protocol            int                     `json:"protocol"`
	Debug               int                     `json:"debug"`
}

// get a config
func NewConfig(file string) (*Config, error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Printf("ioutil.ReadFile(\"%s\") failed (%s)", file, err.Error())
		return nil, err
	}

	cf := &Config{
		Node:      "gopush2-1",
		Addr:      "localhost",
		AdminAddr: "localhost",
		//Pprof:               1,
		MessageExpireSec:    10800,  // 3 hour
		ChannelExpireSec:    604800, // 24 * 7 hour
		Log:                 "./gopush.log",
		MaxStoredMessage:    20,
		MaxSubscriberPerKey: 0, // no limit
		MaxProcs:            runtime.NumCPU(),
		TCPKeepAlive:        1,
		ChannelBucket:       16,
		ChannelType:         0,
		HeartbeatSec:        30,
		Auth:                1,
		Redis:               nil,
		ReadBufInstance:     runtime.NumCPU(),
		ReadBufNumPerInst:   1024,
		ReadBufByte:         512,
		WriteBufNum:         1024,
		WriteBufByte:        512,
		Protocol:            0,
		LogLevel:            0,
		Debug:               0,
	}

	if err = json.Unmarshal(c, cf); err != nil {
		logi.Printf("json.Unmarshal() failed (%s), config json: \"%s\"", err.Error(), string(c))
		return nil, err
	}

	return cf, nil
}
