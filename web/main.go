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
	Log, err = log.New(Conf.LogPath, Conf.LogLevel)
	if err != nil {
		panic(err)
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

	// start pprof http
	StartPprof()

	// External service handle
	http.HandleFunc("/server/get", ServerGet)
	http.HandleFunc("/msg/get", MsgGet)

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
		if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
			Log.Error("http.ListenAndServe(\"%s\") failed(%v)", Conf.Addr, err)
			os.Exit(-1)
		}
	}()

	// init signals, block wait signals
	Log.Info("Web service start")
	HandleSignal(signalCH)

	// Clost message service client
	MsgSvrClose()
	// Stop watch
	WatchStop()
	Log.Warn("Web service end")
}
