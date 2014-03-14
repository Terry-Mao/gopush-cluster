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
	"github.com/Terry-Mao/gopush-cluster/log"
	"os"
	"runtime"
	"time"
)

var (
	Log = log.DefaultLogger
)

func main() {
	var err error
	// parse cmd-line arguments
	flag.Parse()
	signalCH := InitSignal()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"%s\") error(%v)", ConfFile, err)
		os.Exit(-1)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		Log.Error("log.New(\"%s\", %s) error(%v)", Conf.LogFile, Conf.LogLevel, err)
		os.Exit(-1)
	}
	// if process exit, close log
	defer Log.Close()
	Log.Info("comet start")
	// create channel
	UserChannel = NewChannelList()
	// if process exit, close channel
	defer UserChannel.Close()
	// start stats
	StartStats()
	// start pprof http
	StartPprof()
	// init message rpc, block until message rpc init.
	InitMessageRPC()
	// start rpc
	StartRPC()
	// start comet
	StartComet()
	// init zookeeper
	zk, err := InitZookeeper()
	if err != nil {
		Log.Error("InitZookeeper() error(%v)", err)
		os.Exit(-1)
	}
	// if process exit, close zk
	defer zk.Close()
	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() error(%v)", err)
		os.Exit(-1)
	}
	// init signals, block wait signals
	HandleSignal(signalCH)
	// exit
	Log.Info("comet stop")
}
