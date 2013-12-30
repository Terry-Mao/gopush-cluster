package main

import (
	"flag"
	"github.com/Terry-Mao/gopush-cluster/log"
	"os"
	"runtime"
	"runtime/debug"
)

var (
	Log *log.Logger
)

func main() {
	var err error
	defer recoverFunc()
	// parse cmd-line arguments
	flag.Parse()
	// init config
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		log.DefaultLogger.Error("NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		os.Exit(-1)
	}

	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProcs)
	// init log
	if Log, err = log.New(Conf.Log, Conf.LogLevel); err != nil {
		log.DefaultLogger.Error("log.New(\"%s\", %d) failed (%s)", Conf.Log, Conf.LogLevel, err.Error())
		os.Exit(-1)
	}

	defer Log.Close()
	if Conf.Addr == Conf.AdminAddr {
		Log.Warn("Configure \"AdminAdd\" = \"Addr\" is not allowed for security reason")
		os.Exit(-1)
	}

	Log.Info("gopush2 start")
	if err = InitZookeeper(); err != nil {
		Log.Error("InitZookeeper() failed (%s)", err.Error())
	}

	// create channel
	UserChannel = NewChannelList()
	// start stats
	//StartStats()
	// start admin http
	go func() {
		if err := StartAdminHttp(); err != nil {
			Log.Error("StartAdminHttp() failed (%s)", err.Error())
			os.Exit(-1)
		}
	}()

	if Conf.Protocol == WebsocketProtocol {
		// Start http push service
		if err = StartHttp(); err != nil {
			Log.Error("StartHttp() failed (%s)", err.Error())
		}
	} else if Conf.Protocol == TCPProtocol {
		// Start http push service
		if err = StartTCP(); err != nil {
			Log.Error("StartTCP() failed (%s)", err.Error())
		}
	} else {
		Log.Warn("unknown gopush-cluster protocol %d, (0: websocket, 1: tcp)", Conf.Protocol)
		os.Exit(-1)
	}

	// exit
	Log.Info("gopush2 stop")
	os.Exit(0)
}

// recoverFunc log the stack when panic
func recoverFunc() {
	if err := recover(); err != nil {
		Log.Error("panic: (%s)", string(debug.Stack()))
	}
}
