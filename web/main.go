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
	"net"
	"net/http"
	"os"
	"runtime"
	"time"
)

var (
	Log = log.DefaultLogger
)

const (
	httpReadTimeout = 30 //seconds
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
	if Log, err = log.New(Conf.LogPath, Conf.LogLevel); err != nil {
		panic(err)
		os.Exit(-1)
	}
	Log.Debug("initialize config %v", *Conf)

	// Initialize zookeeper
	if err := InitWatch(); err != nil {
		Log.Error("InitWatch() failed(%v)", err)
		os.Exit(-1)
	}

	// Begin watch nodes
	if err := BeginWatchNode(); err != nil {
		Log.Error("BeginWatchNode() failed(%v)", err)
		os.Exit(-1)
	}

	// Initialize message server client
	if err := InitMsgSvrClient(); err != nil {
		Log.Error("InitMsgSvrClient() failed(%v)", err)
		os.Exit(-1)
	}

	// start pprof http
	perf.Init(Conf.PprofBind)

	// Internal admin handle
	go func() {
		adminServeMux := http.NewServeMux()

		adminServeMux.HandleFunc("/admin/push", AdminPushPrivate)
		adminServeMux.HandleFunc("/admin/push/public", AdminPushPublic)
		adminServeMux.HandleFunc("/admin/node/add", AdminNodeAdd)
		adminServeMux.HandleFunc("/admin/node/del", AdminNodeDel)
		adminServeMux.HandleFunc("/admin/msg/clean", AdminMsgClean)

		err := http.ListenAndServe(Conf.AdminAddr, adminServeMux)
		if err != nil {
			Log.Error("http.ListenAndServe(\"%s\") failed(%v)", Conf.AdminAddr, err)
			os.Exit(-1)
		}
	}()

	// Start service
	go func() {
		// External service handle
		httpServeMux := http.NewServeMux()
		httpServeMux.HandleFunc("/server/get", ServerGet)
		httpServeMux.HandleFunc("/msg/get", MsgGet)
		httpServeMux.HandleFunc("/time/get", TimeGet)

		server := &http.Server{Handler: httpServeMux, ReadTimeout: httpReadTimeout * time.Second}
		l, err := net.Listen("tcp", Conf.Addr)
		if err != nil {
			Log.Error("net.Listen(\"tcp\", \"%s\") error(%v)", Conf.Addr, err)
			os.Exit(-1)
		}
		if err := server.Serve(l); err != nil {
			Log.Error("server.Serve(\"%s\") error(%v)", Conf.Addr, err)
			os.Exit(-1)
		}
	}()

	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = process.Init(Conf.User, Conf.Dir, Conf.PidFile); err != nil {
		Log.Error("process.Init() error(%v)", err)
		os.Exit(-1)
	}

	// init signals, block wait signals
	Log.Info("Web service start")
	HandleSignal(signalCH)

	// Clost message service client
	MsgSvrClose()
	// Stop watch
	WatchStop()
	Log.Warn("Web service end")
}
