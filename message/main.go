package main

import (
	"flag"
	"github.com/Terry-Mao/gopush-cluster/log"
	"os"
	"runtime"
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
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() error(%v)", err)
		os.Exit(-1)
	}

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
