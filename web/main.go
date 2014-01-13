package main

import (
	"flag"
	"github.com/Terry-Mao/gopush-cluster/log"
	"net/http"
	"os"
	"runtime"
)

var (
	Log *log.Logger
)

func main() {
	var err error

	runtime.GOMAXPROCS(runtime.NumCPU())

	// Load config
	InitConfig()
	flag.Parse()
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"ConfigPath\":%s) failed(%v)", ConfFile, err)
		os.Exit(-1)
	}

	// Load log
	Log, err = log.New(Conf.LogPath, Conf.LogLevel)
	if err != nil {
		Log.Error("log.New(\"LogPath\":%s) failed(%v)", Conf.LogPath, err)
		os.Exit(-1)
	}

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

	// External service handle
	http.HandleFunc("/server/get", ServerGet)
	http.HandleFunc("/msg/get", MsgGet)

	// Internal admin handle
	go func() {
		adminServeMux := http.NewServeMux()

		adminServeMux.HandleFunc("/admin/push", AdminPush)
		adminServeMux.HandleFunc("/admin/node/add", AdminNodeAdd)
		adminServeMux.HandleFunc("/admin/node/del", AdminNodeDel)

		err := http.ListenAndServe(Conf.AdminAddr, adminServeMux)
		if err != nil {
			panic(err)
		}
	}()

	// Start service
	if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
		Log.Error("http.ListenAndServe(%s) failed(%v)", Conf.Addr, err)
		os.Exit(-1)
	}
}
