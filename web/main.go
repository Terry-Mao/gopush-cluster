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
	var err error
	// Parse cmd-line arguments
	flag.Parse()
	log.Info("web ver: \"%s\" start", ver.Version)
	if err = InitConfig(); err != nil {
		panic(err)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init log
	log.LoadConfiguration(Conf.Log)
	defer log.Close()
	// init zookeeper
	zkConn, err := InitZK()
	if err != nil {
		panic(err)
	}
	// if process exit, close zk
	defer zkConn.Close()
	// start pprof http
	perf.Init(Conf.PprofBind)
	// Init network router
	if Conf.Router != "" {
		if err := InitRouter(); err != nil {
			panic(err)
		}
	}
	// start http listen.
	StartHTTP()
	// create pid file
	if err := ioutil.WriteFile(Conf.PidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644); err != nil {
		panic(err)
	}
	// init signals, block wait signals
	signalCH := InitSignal()
	HandleSignal(signalCH)
	log.Info("web stop")
}
