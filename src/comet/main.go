package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/debug"
)

func main() {
	var err error
	defer recoverFunc()

	// parse cmd-line arguments
	flag.Parse()
	// init config
	Conf, err = NewConfig(ConfFile)
	if err != nil {
		LogError(LogLevelErr, "NewConfig(\"%s\") failed (%s)", ConfFile, err.Error())
		os.Exit(-1)
	}

	// Set max routine
	runtime.GOMAXPROCS(Conf.MaxProcs)
	// init log
	if err = NewLog(); err != nil {
		LogError(LogLevelErr, "NewLog() failed (%s)", err.Error())
		os.Exit(-1)
	}

	defer CloseLog()
	LogError(LogLevelInfo, "gopush2 start")
	// create channel
	if channel = NewChannelList(); channel == nil {
		LogError(LogLevelWarn, "NewChannelList() failed, can't create channellist")
		os.Exit(-1)
	}

	// start stats
	StartStats()
	if Conf.Addr == Conf.AdminAddr {
		LogError(LogLevelWarn, "\"AdminAdd = Addr\" is not allowed for security reason")
		os.Exit(-1)
	}

	// start admin http
	go func() {
		if err := StartAdminHttp(); err != nil {
			LogError(LogLevelErr, "StartAdminHttp() failed (%s)", err.Error())
			os.Exit(-1)
		}
	}()

	if Conf.Protocol == WebsocketProtocol {
		// Start http push service
		if err = StartHttp(); err != nil {
			LogError(LogLevelErr, "StartHttp() failed (%s)", err.Error())
		}
	} else if Conf.Protocol == TCPProtocol {
		// Start http push service
		if err = StartTCP(); err != nil {
			LogError(LogLevelErr, "StartTCP() failed (%s)", err.Error())
		}
	} else {
		LogError(LogLevelWarn, "not support gopush2 protocol %d, (0: websocket, 1: tcp)", Conf.Protocol)
		os.Exit(-1)
	}

	LogError(LogLevelInfo, "gopush2 stop")
}

func recoverFunc() {
	if err := recover(); err != nil {
		LogError(LogLevelErr, "Error : %v, Debug : \n%s", err, string(debug.Stack()))
	}
}
