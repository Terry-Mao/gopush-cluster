package main

import (
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net/rpc"
	"time"
)

var (
	MsgSvrClient *rpc.Client
)

func InitMsgSvrClient() error {
	var err error
	MsgSvrClient, err = rpc.Dial(Conf.MsgSvr.Network, Conf.MsgSvr.Addr)
	if err != nil {
		return err
	}

	// rpc Ping
	go func() {
		for {
			reply := 0
			if err := MsgSvrClient.Call("MessageRPC.Ping", 0, &reply); err != nil {
				Log.Error("rpc.Call(\"MessageRPC.Ping\") failed (%v)", err)
				rpcTmp, err := rpc.Dial("tcp", Conf.Addr)
				if err != nil {
					Log.Error("rpc.Dial(\"tcp\", %s) failed (%v)", Conf.Addr, err)
					time.Sleep(time.Duration(Conf.MsgSvr.Retry) * time.Second)
					Log.Warn("rpc reconnect \"%s\" after %d second", Conf.Addr, Conf.MsgSvr.Retry)
				} else {
					Log.Info("rpc client reconnect \"%s\" ok", Conf.Addr)
					MsgSvrClient = rpcTmp
				}

				continue
			}

			// every one second send a heartbeat ping
			Log.Debug("rpc ping ok")
			time.Sleep(time.Duration(Conf.MsgSvr.Heartbeat) * time.Second)
		}
	}()

	return nil
}

func MessageRPCGet(mid int64, key string) (*myrpc.MessageGetResp, error) {
	var reply myrpc.MessageGetResp
	args := &myrpc.MessageGetArgs{MsgID: mid, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Get", args, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}
