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
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/golang/glog"
	"net"
	"net/rpc"
	"time"
)

var (
	// Delete messages channel
	DelChan chan *DelMessageInfo
)

// RPC For receive offline messages
type MessageRPC struct {
}

// StartRPC start accept rpc call
func StartRPC() {
	DelChan = make(chan *DelMessageInfo, 10000)
	msg := &MessageRPC{}
	rpc.Register(msg)
	for _, bind := range Conf.Addr {
		glog.Infof("start rpc listen addr:\"%s\"", bind)
		go rpcListen(bind)
	}
	// Start a routine for delete message
	go DelProc()
}

func rpcListen(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			glog.Errorf("listener.Close() error(%v)", err)
		}
	}()
	rpc.Accept(l)
}

// Store offline pravite message interface
func (r *MessageRPC) Save(m *myrpc.MessageSaveArgs, ret *int) error {
	glog.Infof("save pravite message (mid:%d,msg:%s,expire:%d,key:%s)", m.MsgID, m.Msg, m.Expire, m.Key)
	if m == nil || m.MsgID < 0 {
		*ret = myrpc.ParamErr
		return nil
	}

	// Json.Marshal and save the message
	recordMsg := &Message{Msg: m.Msg, Expire: m.Expire, MsgID: m.MsgID}
	if err := UseStorage.Save(m.Key, recordMsg, m.MsgID); err != nil {
		glog.Errorf("UseStorage.Save(\"%s\",\"%v\",\"%d\") error(%v)", m.Key, *recordMsg, m.MsgID, err)
		*ret = myrpc.InternalErr
		return nil
	}

	*ret = myrpc.OK
	return nil
}

// Store offline public message interface
func (r *MessageRPC) SavePub(m *myrpc.MessageSavePubArgs, ret *int) error {
	glog.Infof("save public message (mid:%d,msg:%s,expire:%d,key:%s)", m.MsgID, m.Msg, m.Expire, Conf.PKey)
	if m == nil || m.MsgID < 0 {
		*ret = myrpc.ParamErr
		return nil
	}

	// Json.Marshal and save the message
	recordMsg := &Message{Msg: m.Msg, Expire: m.Expire, MsgID: m.MsgID}
	if err := UseStorage.Save(Conf.PKey, recordMsg, m.MsgID); err != nil {
		glog.Errorf("UseStorage.Save(\"%s\",\"%v\",\"%d\") error(%v)", Conf.PKey, *recordMsg, m.MsgID, err)
		*ret = myrpc.InternalErr
		return nil
	}

	*ret = myrpc.OK
	return nil
}

// Get offline message interface
func (r *MessageRPC) Get(m *myrpc.MessageGetArgs, rw *myrpc.MessageGetResp) error {
	glog.Infof("request message (mid:%d,pmid:%d,key:%s)", m.MsgID, m.PubMsgID, m.Key)
	// Get all of offline messages which larger than MsgID that corresponding to m.Key
	msgs, err := UseStorage.Get(m.Key, m.MsgID)
	if err != nil {
		glog.Errorf("UseStorage.Get(\"%s\", \"%d\") error(%v)", m.Key, m.MsgID, err)
		rw.Ret = myrpc.InternalErr
		return nil
	}

	// Get public offline messages which larger than PubMsgID
	pMsgs, err := UseStorage.Get(Conf.PKey, m.PubMsgID)
	if err != nil {
		glog.Errorf("UseStorage.Get(\"%s\", \"%d\") error(%v)", Conf.PKey, m.PubMsgID, err)
		rw.Ret = myrpc.InternalErr
		return nil
	}

	numMsg := len(msgs)
	numPMsg := len(pMsgs)
	if numMsg == 0 && numPMsg == 0 {
		rw.Ret = myrpc.OK
		glog.Infof("response message nil, request key(\"%s\") mid(\"%d\") pmid(\"%d\")", m.Key, m.MsgID, m.PubMsgID)
		return nil
	}

	var (
		data     []string
		pData    []string
		delMsgs  []string
		delPMsgs []string
		msg      = &Message{}
		tNow     = time.Now().Unix()
	)

	// Checkout expired offline messages
	for i := 0; i < numMsg; i++ {
		if err := json.Unmarshal([]byte(msgs[i]), &msg); err != nil {
			glog.Errorf("internal message:\"%s\" error(%v)", msgs[i], err)
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
			glog.Errorf("internal message:\"%s\" error(%v)", pMsgs[i], err)
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
	// TODO:delete expired message is useless as far as mysql storage
	if len(delMsgs) != 0 {
		glog.Infof("delete expire private messages:\"%s\"", msgs)
		DelChan <- &DelMessageInfo{Key: m.Key, Msgs: delMsgs}
	}
	if len(delPMsgs) != 0 {
		glog.Infof("delete expire public messages:\"%s\"", pMsgs)
		DelChan <- &DelMessageInfo{Key: Conf.PKey, Msgs: delPMsgs}
	}

	rw.Ret = myrpc.OK
	rw.Msgs = data
	rw.PubMsgs = pData
	glog.Infof("response private_message(%s) public_message(%s)", data, pData)

	return nil
}

// Clean offline message interface
func (r *MessageRPC) CleanKey(key string, ret *int) error {
	if err := UseStorage.DelKey(key); err != nil {
		glog.Errorf("clean offline message key:\"%s\" error(%v)", key, err)
		*ret = myrpc.InternalErr
		return nil
	}
	glog.Infof("Clean Offline message key:\"%s\" OK", key)

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
		if err := UseStorage.DelMulti(info); err != nil {
			glog.Errorf("UseStorage.DelMulti(key:\"%s\", Msgs:\"%s\") error(%v)", info.Key, info.Msgs, err)
		}
		glog.Infof("UseStorage.DelMulti(key:\"%s\", Msgs:\"%s\") OK", info.Key, info.Msgs)
	}
}
