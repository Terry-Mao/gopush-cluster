// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

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
		failed := false
		// If process exit, then close Message rpc
		defer func() {
			if MsgSvrClient != nil {
				if err := MsgSvrClient.Close(); err != nil {
					Log.Error("MsgSvrClient.Close() error(%v)", err)
				}
			}
		}()
		for {
			if !failed && MsgSvrClient != nil {
				reply := 0
				if err := MsgSvrClient.Call("MessageRPC.Ping", 0, &reply); err != nil {
					Log.Error("rpc.Call(\"MessageRPC.Ping\")  error(%v)", err)
					failed = true
				} else {
					// every one second send a heartbeat ping
					failed = false
					Log.Debug("rpc ping ok")
					time.Sleep(Conf.MsgPing)
					continue
				}
			}
			// reconnect(init) message rpc
			rpcTmp, err := rpc.Dial("tcp", Conf.MsgAddr)
			if err != nil {
				Log.Error("rpc.Dial(\"tcp\", \"%s\") error(%v), reconnect retry after \"%d\" second", Conf.MsgAddr, err, int64(Conf.MsgRetry)/int64(time.Second))
				time.Sleep(Conf.MsgRetry)
				continue
			}
			MsgSvrClient = rpcTmp
			failed = false
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

// MessageRPCCleanKey message-service clean all offline message of specified key RPC insterface
func MessageRPCCleanKey(key string) (int, error) {
	reply := OK
	if err := MsgSvrClient.Call("MessageRPC.CleanKey", key, &reply); err != nil {
		return InternalErr, err
	}

	return reply, nil
}
