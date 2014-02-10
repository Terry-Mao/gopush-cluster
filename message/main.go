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
	// parse cmd-line arguments
	flag.Parse()
	signalCH := InitSignal()
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		Log.Error("InitConfig() error(%v)", err)
		os.Exit(-1)
	}

	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init process
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() error(%v)", err)
		os.Exit(-1)
	}

	// Load log
	Log, err = log.New(Conf.LogFile, Conf.LogLevel)
	if err != nil {
		Log.Error("log.New(\"%s\") error(%v)", Conf.LogFile, err)
		os.Exit(-1)
	}

	// Initialize redis
	InitRedis()

	// Start rpc
	Log.Info("message start")
	go StartRPC()

	// init signals, block wait signals
	HandleSignal(signalCH)

	// exit
	Log.Info("message stop")
}
