package main

import (
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net/rpc"
	"time"
)

var (
	MsgSvrClient *rpc.Client
)

// InitMsgSvrClient initialize message service client
func InitMsgSvrClient() error {
	go func() {
		var err error
		// If process exit, then close Message rpc
		defer func() {
			if MsgSvrClient != nil {
				if err := MsgSvrClient.Close(); err != nil {
					Log.Error("MsgSvrClient.Close() error(%v)", err)
				}
			}
		}()
		for {
			if MsgSvrClient != nil {
				reply := 0
				if err := MsgSvrClient.Call("MessageRPC.Ping", 0, &reply); err != nil {
					Log.Error("rpc.Call(\"MessageRPC.Ping\")  error(%v)", err)
				} else {
					// every one second send a heartbeat ping
					Log.Debug("rpc ping ok")
					time.Sleep(Conf.MsgPing)
					continue
				}
			}
			// reconnect(init) message rpc
			if MsgSvrClient, err = rpc.Dial("tcp", Conf.MsgAddr); err != nil {
				Log.Error("rpc.Dial(\"tcp\", \"%s\") error(%v), reconnect retry after \"%d\" second", Conf.MsgAddr, err, int64(Conf.MsgRetry)/int64(time.Second))
				time.Sleep(Conf.MsgRetry)
				continue
			}
			Log.Info("rpc client reconnect \"%s\" ok", Conf.MsgAddr)
		}
	}()

	return nil
}

// MsgSvrClose close message service client
func MsgSvrClose() {
	if MsgSvrClient != nil {
		if err := MsgSvrClient.Close(); err != nil {
			Log.Error("MsgSvrClient.Close() error(%v)", err)
		}
	}
}

// Message service message-service get private and public messages RPC insterface
func MessageRPCGet(key string, mid, pMid int64) (*myrpc.MessageGetResp, error) {
	var reply myrpc.MessageGetResp
	args := &myrpc.MessageGetArgs{MsgID: mid, PubMsgID: pMid, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Get", args, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}

// MessageRPCSave message-service save private message RPC insterface
func MessageRPCSave(key, msg string, mid, expire int64) (int, error) {
	reply := OK
	args := &myrpc.MessageSaveArgs{MsgID: mid, Msg: msg, Expire: expire, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Save", args, &reply); err != nil {
		return InternalErr, err
	}

	return reply, nil
}

// MessageRPCSavePub message-service save public message RPC insterface
func MessageRPCSavePub(msg string, mid, expire int64) (int, error) {
	reply := OK
	args := &myrpc.MessageSavePubArgs{MsgID: mid, Msg: msg, Expire: expire}
	if err := MsgSvrClient.Call("MessageRPC.SavePub", args, &reply); err != nil {
		return InternalErr, err
	}

	return reply, nil
}
