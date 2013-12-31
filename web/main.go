package main

import (
	//	"fmt"
	"github.com/Terry-Mao/gopush-cluster/log"
	"net/http"
	"runtime"
)

var (
	Log *log.Logger
)

func init() {
	var err error

	runtime.GOMAXPROCS(runtime.NumCPU())

	//Load config
	initConfig()
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		panic(err)
	}

	//Load log
	Log, err = log.New(Conf.LogPath, log.Debug)
	if err != nil {
		panic(err)
	}

	// Init zookeeper
	if err := initZK(); err != nil {
		panic(err)
	}

	// Init redis
	initRedis()
}

func main() {
	if err := BeginWatchNode(); err != nil {
		panic(err)
	}

	http.HandleFunc("/server/get", ServerGet)
	http.HandleFunc("/msg/get", MsgGet)

	go func() {
		internalServeMux := http.NewServeMux()
		internalServeMux.HandleFunc("/msg/set", MsgSet)
		err := http.ListenAndServe(Conf.InternalAddr, internalServeMux)
		if err != nil {
			panic(err)
		}
	}()

	if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
		panic(err)
	}
}
