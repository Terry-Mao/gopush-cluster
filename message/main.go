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
	"github.com/Terry-Mao/gopush-cluster/perf"
	"github.com/Terry-Mao/gopush-cluster/process"
	"runtime"
	"time"
)

var (
	Log = log.DefaultLogger
)

func main() {
	var err error
	// Parse cmd-line arguments
	flag.Parse()
	signalCH := InitSignal()
	// Load config
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"%s\") error(%v)", ConfFile, err)
		return
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// Load log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		Log.Error("log.New(\"%s\", %s) error(%v)", Conf.LogFile, Conf.LogLevel, err)
		return
	}
	// start pprof http
	perf.Init(Conf.PprofBind)
	// Initialize redis
	InitStorage()
	// Start rpc
	StartRPC()
	// init zookeeper
	zkConn, err := InitZK()
	if err != nil {
		Log.Error("InitZookeeper() error(%v)", err)
		return
	}
	// if process exit, close zk
	defer zkConn.Close()
	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = process.Init(Conf.User, Conf.Dir, Conf.PidFile); err != nil {
		Log.Error("process.Init() error(%v)", err)
		return
	}
	Log.Info("message start")
	// init signals, block wait signals
	HandleSignal(signalCH)
	// exit
	Log.Info("message stop")
}
