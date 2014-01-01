package main

import (
	"flag"
	"fmt"
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

	//Load config
	InitConfig()
	flag.Parse()
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"ConfigPath\":%s) failed(%v)", ConfFile, err)
		os.Exit(-1)
	}
	fmt.Println(Conf)

	//Load log
	Log, err = log.New(Conf.LogPath, log.Debug)
	if err != nil {
		Log.Error("InitZK(\"LogPath\":%s) failed(%v)", Conf.LogPath, err)
		os.Exit(-1)
	}

	// Init zookeeper
	if err := InitZK(); err != nil {
		Log.Error("InitZK() failed(%v)", err)
		os.Exit(-1)
	}

	// Init redis
	InitRedis()

	if err := BeginWatchNode(); err != nil {
		Log.Error("BeginWatchNode() failed(%v)", err)
		os.Exit(-1)
	}

	http.HandleFunc("/server/get", ServerGet)
	http.HandleFunc("/msg/get", MsgGet)

	go func() {
		// Start internal service
		internalServeMux := http.NewServeMux()
		internalServeMux.HandleFunc("/msg/set", MsgSet)
		err := http.ListenAndServe(Conf.InternalAddr, internalServeMux)
		if err != nil {
			Log.Error("http.ListenAndServe(%s) failed(%v)", Conf.InternalAddr, err)
			os.Exit(-1)
		}
	}()

	// Start external service
	if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
		Log.Error("http.ListenAndServe(%s) failed(%v)", Conf.Addr, err)
		os.Exit(-1)
	}
}
