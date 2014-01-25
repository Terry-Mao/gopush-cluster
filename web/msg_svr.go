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
	MsgSvrClient, err = rpc.Dial(Conf.MsgNetwork, Conf.MsgAddr)
	if err != nil {
		return err
	}

	// rpc Ping
	go func() {
		for {
			reply := 0
			if err := MsgSvrClient.Call("MessageRPC.Ping", 0, &reply); err != nil {
				Log.Error("rpc.Call(\"MessageRPC.Ping\") failed (%v)", err)
				rpcTmp, err := rpc.Dial("tcp", Conf.MsgAddr)
				if err != nil {
					Log.Error("rpc.Dial(\"tcp\", %s) failed (%v)", Conf.MsgAddr, err)
					time.Sleep(Conf.MsgRetry)
					Log.Warn("rpc reconnect \"%s\" after %s", Conf.MsgAddr, Conf.MsgRetry)
				} else {
					Log.Info("rpc client reconnect \"%s\" ok", Conf.MsgAddr)
					MsgSvrClient = rpcTmp
				}

				continue
			}

			// every one second send a heartbeat ping
			Log.Debug("rpc ping ok")
			time.Sleep(Conf.MsgHeartbeat)
		}
	}()

	return nil
}

func MessageRPCGet(key string, mid, pMid int64) (*myrpc.MessageGetResp, error) {
	var reply myrpc.MessageGetResp
	args := &myrpc.MessageGetArgs{MsgID: mid, PubMsgID: pMid, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Get", args, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}

func MessageRPCSave(key, msg string, mid, expire int64) (int, error) {
	reply := OK
	args := &myrpc.MessageSaveArgs{MsgID: mid, Msg: msg, Expire: expire, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Save", args, &reply); err != nil {
		return InternalErr, err
	}

	return reply, nil
}
