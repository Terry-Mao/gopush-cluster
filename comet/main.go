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
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		os.Exit(-1)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init process
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() failed (%s)", err.Error())
		os.Exit(-1)
	}
	// init log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		Log.Error("log.New(\"%s\", %s) failed (%s)", Conf.LogFile, Conf.LogLevel, err.Error())
		os.Exit(-1)
	}
	// if process exit, close log
	defer Log.Close()
	Log.Info("gopush2 start")
	zk, err := InitZookeeper()
	if err != nil {
		Log.Error("InitZookeeper() failed (%s)", err.Error())
		os.Exit(-1)
	}
	// if process exit, close zk
	defer zk.Close()
	// create channel
	UserChannel = NewChannelList()
	defer UserChannel.Close()
	// if process exit, close channel
	// start stats
	StartStats()
	// start pprof http
	StartPprof()
	// start comet
	StartComet()
	// init message rpc
	InitMessageRPC()
	// start rpc
	StartRPC()
	// init signals, then block wait signals
	InitSignal()
	// exit
	Log.Info("gopush2 stop")
}
