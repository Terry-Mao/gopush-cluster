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
	log "code.google.com/p/log4go"
	"flag"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/perf"
	"github.com/Terry-Mao/gopush-cluster/ver"
	"io/ioutil"
	"os"
	"runtime"
)

func main() {
	// parse cmd-line arguments
	flag.Parse()
	log.Info("comet ver: \"%s\" start", ver.Version)
	// init config
	if err := InitConfig(); err != nil {
		log.Error("InitConfig() error(%v)", err)
		return
	}
	// set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init log
	log.LoadConfiguration(Conf.Log)
	defer log.Close()
	// start pprof
	perf.Init(Conf.PprofBind)
	// create channel
	// if process exit, close channel
	UserChannel = NewChannelList()
	defer UserChannel.Close()
	// start stats
	StartStats()
	// start rpc
	if err := StartRPC(); err != nil {
		panic(err)
	}
	// start comet
	if err := StartComet(); err != nil {
		panic(err)
	}
	// init zookeeper
	zkConn, err := InitZK()
	if err != nil {
		if zkConn != nil {
			zkConn.Close()
		}
		panic(err)
	}
	// create pid file
	if err := ioutil.WriteFile(Conf.PidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644); err != nil {
		panic(err)
	}
	// init signals, block wait signals
	signalCH := InitSignal()
	HandleSignal(signalCH)
	// exit
	log.Info("comet stop")
}
