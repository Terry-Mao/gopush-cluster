package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/log"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	Log = log.DefaultLogger
)

func main() {
	var err error
	flag.Parse()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		os.Exit(-1)
	}
	// init log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		Log.Error("log.New(\"%s\", %s) failed (%s)", Conf.LogFile, Conf.LogLevel, err.Error())
		os.Exit(-1)
	}
	defer Log.Close()
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		Log.Error("net.ResolveTCPAddr(\"tcp\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		os.Exit(-1)
	}
	Log.Info("connect to gopush-cluster comet")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		Log.Error("net.DialTCP() failed (%s)", err.Error())
		os.Exit(-1)
	}
	Log.Info("send sub request")
	proto := []byte(fmt.Sprintf("*3\r\n$3\r\nsub\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(Conf.Key), Conf.Key, len(strconv.Itoa(int(Conf.Heartbeat))), Conf.Heartbeat))
	Log.Info("send protocol: %s", string(proto))
	if _, err := conn.Write(proto); err != nil {
		Log.Error("conn.Write() failed (%s)", err.Error())
		os.Exit(-1)
	}
	// get first heartbeat
	first := false
	rd := bufio.NewReader(conn)
	// block read reply from service
	Log.Info("wait message")
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(Conf.Heartbeat) * 2)); err != nil {
			Log.Error("conn.SetReadDeadline() failed (%s)", err.Error())
			os.Exit(-1)
		}
		line, err := rd.ReadBytes('\n')
		if err != nil {
			Log.Error("rd.ReadBytes() failed (%s)", err.Error())
			os.Exit(-1)
		}
		if line[len(line)-2] != '\r' {
			Log.Error("protocol reply format error")
			os.Exit(-1)
		}
        Log.Info("line: %s", line)
		switch line[0] {
		// reply
		case '$':
			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				Log.Error("protocol reply format error")
				os.Exit(-1)
			}
			data, err := rd.ReadBytes('\n')
			if err != nil {
				Log.Error("protocol reply format error")
				os.Exit(-1)
			}
			if len(data) != cmdSize+2 {
				Log.Error("protocol reply format error: %s", data)
				os.Exit(-1)
			}
			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				Log.Error("protocol reply format error")
				os.Exit(-1)
			}
			reply := string(data[0:cmdSize])
			Log.Info("receive msg: %s", reply)
			break
			// heartbeat
		case '+':
			if !first {
				// send heartbeat
				go func() {
					for {
						Log.Info("send heartbeat")
						if _, err := conn.Write([]byte("h")); err != nil {
							Log.Error("conn.Write() failed (%s)", err.Error())
							os.Exit(-1)
						}
						time.Sleep(time.Duration(Conf.Heartbeat) * time.Second)
					}
				}()
				first = true
			}
			Log.Info("receive heartbeat")
			break
		}
	}
}
