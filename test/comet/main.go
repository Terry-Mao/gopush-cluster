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
	log "code.google.com/p/log4go"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"
)

func main() {
	var err error
	flag.Parse()
	// init config
	Conf, err := InitConfig(ConfFile)
	if err != nil {
		log.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		return
	}
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		log.Error("net.ResolveTCPAddr(\"tcp\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		return
	}
	log.Info("connect to gopush-cluster comet")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Error("net.DialTCP() failed (%s)", err.Error())
		return
	}
	log.Info("send sub request")
	proto := []byte(fmt.Sprintf("*3\r\n$3\r\nsub\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(Conf.Key), Conf.Key, len(strconv.Itoa(int(Conf.Heartbeat))), Conf.Heartbeat))
	log.Info("send protocol: %s", string(proto))
	if _, err := conn.Write(proto); err != nil {
		log.Error("conn.Write() failed (%s)", err.Error())
		return
	}
	// get first heartbeat
	first := false
	rd := bufio.NewReader(conn)
	// block read reply from service
	log.Info("wait message")
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(Conf.Heartbeat) * 2)); err != nil {
			log.Error("conn.SetReadDeadline() failed (%s)", err.Error())
			return
		}
		line, err := rd.ReadBytes('\n')
		if err != nil {
			log.Error("rd.ReadBytes() failed (%s)", err.Error())
			return
		}
		if line[len(line)-2] != '\r' {
			log.Error("protocol reply format error")
			return
		}
		log.Info("line: %s", line)
		switch line[0] {
		// reply
		case '$':
			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				log.Error("protocol reply format error")
				return
			}
			data, err := rd.ReadBytes('\n')
			if err != nil {
				log.Error("protocol reply format error")
				return
			}
			if len(data) != cmdSize+2 {
				log.Error("protocol reply format error: %s", data)
				return
			}
			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				log.Error("protocol reply format error")
				return
			}
			reply := string(data[0:cmdSize])
			log.Info("receive msg: %s", reply)
			break
			// heartbeat
		case '+':
			if !first {
				// send heartbeat
				go func() {
					for {
						log.Info("send heartbeat")
						if _, err := conn.Write([]byte("h")); err != nil {
							log.Error("conn.Write() failed (%s)", err.Error())
							return
						}
						time.Sleep(time.Duration(Conf.Heartbeat) * time.Second)
					}
				}()
				first = true
			}
			log.Info("receive heartbeat")
			break
		}
	}
}
