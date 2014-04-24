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
	"net"
	"net/http"
	"runtime"
	"time"
)

const (
	httpReadTimeout = 30 //seconds
)

func main() {
	var err error
	// Parse cmd-line arguments
	flag.Parse()
	defer glog.Flush()
	signalCH := InitSignal()
	// Load config
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		glog.Errorf("NewConfig(\"%s\") error(%v)", ConfFile, err)
		return
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// Initialize zookeeper
	zk, err := InitZK()
	if err != nil {
		glog.Errorf("InitZK() failed(%v)", err)
		return
	}
	// if process exit, close zk
	defer zk.Close()
	// Initialize message server client
	if err := InitMsgSvrClient(); err != nil {
		glog.Errorf("InitMsgSvrClient() failed(%v)", err)
		return
	}
	// Clost message service client
	defer MsgSvrClose()
	// start pprof http
	perf.Init(Conf.PprofBind)
	// Init network router
	if Conf.Router != "" {
		if err := InitRouter(); err != nil {
			glog.Errorf("InitRouter() failed(%v)", err)
			return
		}
	}

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
			glog.Errorf("http.ListenAndServe(\"%s\") failed(%v)", Conf.AdminAddr, err)
			panic(err)
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
			glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", Conf.Addr, err)
			panic(err)
		}
		if err := server.Serve(l); err != nil {
			glog.Errorf("server.Serve(\"%s\") error(%v)", Conf.Addr, err)
			panic(err)
		}
	}()
	// init process
	// sleep one second, let the listen start
	time.Sleep(time.Second)
	if err = process.Init(Conf.User, Conf.Dir, Conf.PidFile); err != nil {
		glog.Errorf("process.Init() error(%v)", err)
		return
	}
	// init signals, block wait signals
	glog.Infof("Web service start")
	HandleSignal(signalCH)
	glog.Infof("Web service end")
}
