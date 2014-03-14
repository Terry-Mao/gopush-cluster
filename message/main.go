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
	// Parse cmd-line arguments
	flag.Parse()
	signalCH := InitSignal()

	// Load config
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		panic(err)
		os.Exit(-1)
	}

	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)

	// Load log
	Log, err = log.New(Conf.LogFile, Conf.LogLevel)
	if err != nil {
		panic(err)
		os.Exit(-1)
	}

	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() error(%v)", err)
		os.Exit(-1)
	}

	// start pprof http
	StartPprof()

	// Initialize redis
	InitRedis()

	// Start rpc
	Log.Info("Message service start")
	go StartRPC()

	// init signals, block wait signals
	HandleSignal(signalCH)

	// exit
	Log.Info("Message service end")
}
