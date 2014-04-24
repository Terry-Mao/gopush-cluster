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
	"flag"
	"github.com/Terry-Mao/gopush-cluster/perf"
	"github.com/Terry-Mao/gopush-cluster/process"
	"github.com/golang/glog"
	"runtime"
	"time"
)

func main() {
	var err error
	// parse cmd-line arguments
	flag.Parse()
	defer glog.Flush()
	signalCH := InitSignal()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		glog.Errorf("NewConfig(\"%s\") error(%v)", ConfFile, err)
		return
	}
	// set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// create channel
	UserChannel = NewChannelList()
	// if process exit, close channel
	defer UserChannel.Close()
	// start stats
	StartStats()
	// start pprof
	perf.Init(Conf.PprofBind)
	// init message rpc, block until message rpc init.
	InitMessageRPC()
	// start rpc
	StartRPC()
	// start comet
	StartComet()
	// init zookeeper
	zkConn, err := InitZK()
	if err != nil {
		glog.Errorf("InitZookeeper() error(%v)", err)
		return
	}
	// if process exit, close zk
	defer zkConn.Close()
	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = process.Init(Conf.User, Conf.Dir, Conf.PidFile); err != nil {
		glog.Errorf("process.Init() error(%v)", err)
		return
	}
	glog.Info("comet start")
	// init signals, block wait signals
	HandleSignal(signalCH)
	// exit
	glog.Info("comet stop")
}
