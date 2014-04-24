// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"net"
	"strconv"
	"time"
)

func main() {
	var err error
	flag.Parse()
	defer glog.Flush()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		glog.Errorf("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		return
	}
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		glog.Errorf("net.ResolveTCPAddr(\"tcp\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		return
	}
	glog.Info("connect to gopush-cluster comet")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		glog.Errorf("net.DialTCP() failed (%s)", err.Error())
		return
	}
	glog.Infof("send sub request")
	proto := []byte(fmt.Sprintf("*3\r\n$3\r\nsub\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(Conf.Key), Conf.Key, len(strconv.Itoa(int(Conf.Heartbeat))), Conf.Heartbeat))
	glog.Infof("send protocol: %s", string(proto))
	if _, err := conn.Write(proto); err != nil {
		glog.Errorf("conn.Write() failed (%s)", err.Error())
		return
	}
	// get first heartbeat
	first := false
	rd := bufio.NewReader(conn)
	// block read reply from service
	glog.Info("wait message")
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(Conf.Heartbeat) * 2)); err != nil {
			glog.Errorf("conn.SetReadDeadline() failed (%s)", err.Error())
			return
		}
		line, err := rd.ReadBytes('\n')
		if err != nil {
			glog.Errorf("rd.ReadBytes() failed (%s)", err.Error())
			return
		}
		if line[len(line)-2] != '\r' {
			glog.Errorf("protocol reply format error")
			return
		}
		glog.Infof("line: %s", line)
		switch line[0] {
		// reply
		case '$':
			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				glog.Errorf("protocol reply format error")
				return
			}
			data, err := rd.ReadBytes('\n')
			if err != nil {
				glog.Errorf("protocol reply format error")
				return
			}
			if len(data) != cmdSize+2 {
				glog.Errorf("protocol reply format error: %s", data)
				return
			}
			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				glog.Errorf("protocol reply format error")
				return
			}
			reply := string(data[0:cmdSize])
			glog.Infof("receive msg: %s", reply)
			break
			// heartbeat
		case '+':
			if !first {
				// send heartbeat
				go func() {
					for {
						glog.Info("send heartbeat")
						if _, err := conn.Write([]byte("h")); err != nil {
							glog.Errorf("conn.Write() failed (%s)", err.Error())
							return
						}
						time.Sleep(time.Duration(Conf.Heartbeat) * time.Second)
					}
				}()
				first = true
			}
			glog.Info("receive heartbeat")
			break
		}
	}
}
