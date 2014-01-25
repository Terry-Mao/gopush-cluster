package main

import (
	"flag"
	"github.com/Terry-Mao/gopush-cluster/log"
	"os"
	"runtime"
)

var (
	Log *log.Logger
)

func main() {
	var err error
	// parse cmd-line arguments
	flag.Parse()
	// init config
	Conf, err = InitConfig(ConfFile)
	if err != nil {
		log.DefaultLogger.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		os.Exit(-1)
	}
	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProc)
	// init log
	if Log, err = log.New(Conf.LogFile, Conf.LogLevel); err != nil {
		log.DefaultLogger.Error("log.New(\"%s\", %d) failed (%s)", Conf.LogFile, Conf.LogLevel, err.Error())
		os.Exit(-1)
	}
	defer Log.Close()
	// init process
	if err = InitProcess(); err != nil {
		Log.Error("InitProcess() failed (%s)", err.Error())
		os.Exit(-1)
	}
	Log.Info("gopush2 start")
	StartStats()
	if zk, err := InitZookeeper(); err != nil {
		Log.Error("InitZookeeper() failed (%s)", err.Error())
		os.Exit(-1)
	} else {
		defer func() {
			if err = zk.Close(); err != nil {
				Log.Error("zk.Close() failed (%s)", err.Error())
			}
		}()
	}
	// create channel
	UserChannel = NewChannelList()
	// start stats
	StartStats()
	// start pprof http
	StartPprof()
	// start comet
	StartComet()
	// start rpc
	StartRPC()
	// exit
	Log.Info("gopush2 stop")
	os.Exit(-1)
}
