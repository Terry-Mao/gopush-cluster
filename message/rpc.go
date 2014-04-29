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
	"github.com/golang/glog"
	"net"
	"net/rpc"
)

// RPC For receive offline messages
type MessageRPC struct {
}

// InitRPC start accept rpc call.
func InitRPC() {
	msg := &MessageRPC{}
	rpc.Register(msg)
	for _, bind := range Conf.RPCBind {
		glog.Infof("start rpc listen addr: \"%s\"", bind)
		go rpcListen(bind)
	}
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
	if m == nil || m.Msg == nil || m.MsgId < 0 || m.GroupId < 0 {
		return myrpc.ErrParam
	}
	if err := UseStorage.Save(m.Key, m.Msg, m.MsgId, m.GroupId, m.Expire); err != nil {
		glog.Errorf("UseStorage.Save(\"%s\", \"%s\", %d, %d, %d) error(%v)", m.Key, string(m.Msg), m.MsgId, m.GroupId, m.Expire, err)
		return err
	}
	glog.V(1).Infof("UseStorage.Save(\"%s\", \"%s\", %d, %d, %d) ok", m.Key, string(m.Msg), m.MsgId, m.GroupId, m.Expire)
	return nil
}

// Store offline public message interface
func (r *MessageRPC) SavePub(m *myrpc.MessageSavePubArgs, ret *int) error {
	return nil
}

// Get offline message interface
func (r *MessageRPC) Get(m *myrpc.MessageGetArgs, rw *myrpc.MessageGetResp) error {
	if m == nil || m.Key == "" || m.MsgId < 0 {
		return myrpc.ErrParam
	}
	msgs, err := UseStorage.Get(m.Key, m.MsgId)
	if err != nil {
		glog.Errorf("UseStorage.Get(\"%s\", %d) error(%v)", m.Key, m.MsgId, err)
		return err
	}
	rw.Msgs = msgs
	// TODO
	rw.PubMsgs = nil
	glog.V(1).Infof("UserStorage.Get(\"%s\", %d) ok", m.Key, m.MsgId)
	return nil
}

// Clean offline message interface
func (r *MessageRPC) Clean(key string, ret *int) error {
	if key == "" {
		return myrpc.ErrParam
	}
	if err := UseStorage.Del(key); err != nil {
		glog.Errorf("UserStorage.Del(\"%s\") error(%v)", key, err)
		return err
	}
	glog.V(1).Infof("UserStorage.Del(\"%s\") ok", key)
	return nil
}

// Server Ping interface
func (r *MessageRPC) Ping(p int, ret *int) error {
	glog.V(2).Info("ping ok")
	return nil
}
