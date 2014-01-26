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
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		Log.Error("NewConfig(\"%s\") error(%v)", ConfFile, err)
		os.Exit(-1)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init process
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() error(%v)", err)
		os.Exit(-1)
	}
	// init log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		Log.Error("log.New(\"%s\", %s) error(%v)", Conf.LogFile, Conf.LogLevel, err)
		os.Exit(-1)
	}
	// if process exit, close log
	defer Log.Close()
	Log.Info("comet start")
	// create channel
	UserChannel = NewChannelList()
	// if process exit, close channel
	defer UserChannel.Close()
	// start stats
	StartStats()
	// start pprof http
	StartPprof()
	// init message rpc, block until message rpc init.
	InitMessageRPC()
	// start rpc
	StartRPC()
	// start comet
	StartComet()
	// init zookeeper
	zk, err := InitZookeeper()
	if err != nil {
		Log.Error("InitZookeeper() error(%v)", err)
		os.Exit(-1)
	}
	// if process exit, close zk
	defer zk.Close()
	// init signals, block wait signals
	HandleSignal(signalCH)
	// exit
	Log.Info("comet stop")
}
