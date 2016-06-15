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
	log "github.com/alecthomas/log4go"
	"flag"
	"github.com/Terry-Mao/gopush-cluster/perf"
	"github.com/Terry-Mao/gopush-cluster/process"
	"github.com/Terry-Mao/gopush-cluster/ver"
	"runtime"
)

func main() {
	flag.Parse()
	log.Info("message ver: \"%s\" start", ver.Version)
	if err := InitConfig(); err != nil {
		panic(err)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init log
	log.LoadConfiguration(Conf.Log)
	defer log.Close()
	// start pprof http
	perf.Init(Conf.PprofBind)
	// Initialize redis
	if err := InitStorage(); err != nil {
		panic(err)
	}
	// init rpc service
	if err := InitRPC(); err != nil {
		panic(err)
	}
	// init zookeeper
	zk, err := InitZK()
	if err != nil {
		if zk != nil {
			zk.Close()
		}
		panic(err)
	}
	// process init
	if err = process.Init(Conf.User, Conf.Dir, Conf.PidFile); err != nil {
		panic(err)
	}
	// init signals, block wait signals
	sig := InitSignal()
	HandleSignal(sig)
	// exit
	log.Info("message stop")
}
