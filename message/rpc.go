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
	"encoding/json"
	. "github.com/Terry-Mao/gopush-cluster/log"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"net/rpc"
	"os"
	"time"
)

var (
	// Delete messages channel
	DelChan chan *DelMessageInfo
)

// The Message struct
type Message struct {
	// Message
	Msg string `json:"msg"`
	// Message expired unixnano
	Expire int64 `json:"expire"`
	// Message id
	MsgID int64 `json:"mid"`
}

// RPC For receive offline messages
type MessageRPC struct {
}

// StartRPC start accept rpc call
func StartRPC() {
	DelChan = make(chan *DelMessageInfo, 10000)
	msg := &MessageRPC{}
	rpc.Register(msg)

	// Start a routine for delete message
	go DelProc()

	l, err := net.Listen("tcp", Conf.Addr)
	if err != nil {
		Log.Error("net.Listen(\"tcp\", \"%s\") error(%v)", Conf.Addr, err)
		os.Exit(-1)
	}

	defer func() {
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() error(%v)", err)
		}
	}()

	Log.Info("start listen admin addr:\"%s\"", Conf.Addr)
	rpc.Accept(l)
}

// Store offline pravite message interface
func (r *MessageRPC) Save(m *myrpc.MessageSaveArgs, ret *int) error {
	Log.Info("save pravite message (mid:%d,msg:%s,expire:%d,key:%s)", m.MsgID, m.Msg, m.Expire, m.Key)
	if m == nil || m.MsgID < 0 {
		*ret = myrpc.ParamErr
		return nil
	}

	// Json.Marshal and save the message
	recordMsg := Message{Msg: m.Msg, Expire: m.Expire, MsgID: m.MsgID}
	message, _ := json.Marshal(recordMsg)
	if err := SaveMessage(m.Key, string(message), m.MsgID); err != nil {
		Log.Error("save message error(%v)", err)
		*ret = myrpc.InternalErr
		return nil
	}

	*ret = myrpc.OK
	return nil
}

// Store offline public message interface
func (r *MessageRPC) SavePub(m *myrpc.MessageSavePubArgs, ret *int) error {
	Log.Info("save public message (mid:%d,msg:%s,expire:%d,key:%s)", m.MsgID, m.Msg, m.Expire, Conf.PKey)
	if m == nil || m.MsgID < 0 {
		*ret = myrpc.ParamErr
		return nil
	}

	// Json.Marshal and save the message
	recordMsg := Message{Msg: m.Msg, Expire: m.Expire, MsgID: m.MsgID}
	message, _ := json.Marshal(recordMsg)
	if err := SaveMessage(Conf.PKey, string(message), m.MsgID); err != nil {
		Log.Error("save message error(%v)", err)
		*ret = myrpc.InternalErr
		return nil
	}

	*ret = myrpc.OK
	return nil
}

// Get offline message interface
func (r *MessageRPC) Get(m *myrpc.MessageGetArgs, rw *myrpc.MessageGetResp) error {
	Log.Info("request message (mid:%d,pmid:%d,key:%s)", m.MsgID, m.PubMsgID, m.Key)
	// Get all of offline messages which larger than MsgID that corresponding to m.Key
	msgs, err := GetMessages(m.Key, m.MsgID)
	if err != nil {
		Log.Error("get messages error(%v)", err)
		rw.Ret = myrpc.InternalErr
		return nil
	}

	// Get public offline messages which larger than PubMsgID
	pMsgs, err := GetMessages(Conf.PKey, m.PubMsgID)
	if err != nil {
		Log.Error("get public messages error(%v)", err)
		rw.Ret = myrpc.InternalErr
		return nil
	}

	numMsg := len(msgs)
	numPMsg := len(pMsgs)
	if numMsg == 0 && numPMsg == 0 {
		rw.Ret = myrpc.OK
		Log.Info("response message nil, request key(\"%s\") mid(\"%d\") pmid(\"%d\")", m.Key, m.MsgID, m.PubMsgID)
		return nil
	}

	var (
		data     []string
		pData    []string
		delMsgs  []string
		delPMsgs []string
		msg      = &Message{}
		tNow     = time.Now().UnixNano()
	)

	// Checkout expired offline messages
	for i := 0; i < numMsg; i++ {
		if err := json.Unmarshal([]byte(msgs[i]), &msg); err != nil {
			Log.Error("internal message:\"%s\" error(%v)", msgs[i], err)
			rw.Ret = myrpc.InternalErr
			return nil
		}

		if tNow > msg.Expire {
			delMsgs = append(delMsgs, msgs[i])
			continue
		}

		data = append(data, msgs[i])
	}
	for i := 0; i < numPMsg; i++ {
		if err := json.Unmarshal([]byte(pMsgs[i]), &msg); err != nil {
			Log.Error("internal message:\"%s\" error(%v)", pMsgs[i], err)
			rw.Ret = myrpc.InternalErr
			return nil
		}
		if tNow > msg.Expire {
			delPMsgs = append(delPMsgs, pMsgs[i])
			continue
		}
		pData = append(pData, pMsgs[i])
	}

	// Send to delete message process
	if len(delMsgs) != 0 {
		Log.Info("delete expire private messages:\"%s\"", msgs)
		DelChan <- &DelMessageInfo{Key: m.Key, Msgs: delMsgs}
	}
	if len(delPMsgs) != 0 {
		Log.Info("delete expire public messages:\"%s\"", pMsgs)
		DelChan <- &DelMessageInfo{Key: Conf.PKey, Msgs: delPMsgs}
	}

	rw.Ret = myrpc.OK
	rw.Msgs = data
	rw.PubMsgs = pData
	Log.Info("response private_message(%s) public_message(%s)", data, pData)

	return nil
}

// Clean offline message interface
func (r *MessageRPC) CleanKey(key string, ret *int) error {
	if err := DelKey(key); err != nil {
		Log.Error("clean offline message error(%v)", err)
		*ret = myrpc.InternalErr
		return nil
	}
	Log.Info("Clean Offline message key(\"%s\") OK", key)

	return nil
}

// Server Ping interface
func (r *MessageRPC) Ping(p int, ret *int) error {
	*ret = myrpc.OK
	return nil
}

// Asynchronous delete message
func DelProc() {
	for {
		info := <-DelChan
		if err := DelMessages(info); err != nil {
			Log.Error("DelMessages(key:\"%s\", Msgs:\"%s\") error(%v)", info.Key, info.Msgs, err)
		}
		Log.Info("DelMessages(key:\"%s\", Msgs:\"%s\") OK", info.Key, info.Msgs)
	}
}
