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

// SavePrivate rpc interface save user private message.
func (r *MessageRPC) SavePrivate(m *myrpc.MessageSavePrivateArgs, ret *int) error {
	if m == nil || m.Msg == nil || m.MsgId < 0 {
		return myrpc.ErrParam
	}
	if err := UseStorage.SavePrivate(m.Key, m.Msg, m.MsgId, m.Expire); err != nil {
		glog.Errorf("UseStorage.SavePrivate(\"%s\", \"%s\", %d, %d) error(%v)", m.Key, string(m.Msg), m.MsgId, m.Expire, err)
		return err
	}
	glog.V(1).Infof("UseStorage.SavePrivate(\"%s\", \"%s\", %d, %d) ok", m.Key, string(m.Msg), m.MsgId, m.Expire)
	return nil
}

// GetPrivate rpc interface get user private message.
func (r *MessageRPC) GetPrivate(m *myrpc.MessageGetPrivateArgs, rw *myrpc.MessageGetResp) error {
	if m == nil || m.Key == "" || m.MsgId < 0 {
		return myrpc.ErrParam
	}
	msgs, err := UseStorage.GetPrivate(m.Key, m.MsgId)
	if err != nil {
		glog.Errorf("UseStorage.GetPrivate(\"%s\", %d) error(%v)", m.Key, m.MsgId, err)
		return err
	}
	rw.Msgs = msgs
	glog.V(1).Infof("UserStorage.GetPrivate(\"%s\", %d) ok", m.Key, m.MsgId)
	return nil
}

// DelPrivate rpc interface delete user private message.
func (r *MessageRPC) DelPrivate(key string, ret *int) error {
	if key == "" {
		return myrpc.ErrParam
	}
	if err := UseStorage.DelPrivate(key); err != nil {
		glog.Errorf("UserStorage.DelPrivate(\"%s\") error(%v)", key, err)
		return err
	}
	glog.V(1).Infof("UserStorage.DelPrivate(\"%s\") ok", key)
	return nil
}

/*
// SavePublish rpc interface save publish message.
func (r *MessageRPC) SavePublish(m *myrpc.MessageSaveGroupArgs, ret *int) error {
	return nil
}

// GetPublish rpc interface get publish message.
func (r *MessageRPC) GetPublish(m *myrpc.MessageGetGroupArgs, rw *myrpc.MessageGetResp) error {
	return nil
}

// DelPublish rpc interface delete publish message.
func (r *MessageRPC) DelPublish(key string, ret *int) error {
	return nil
}

// SaveGroup rpc interface save publish message.
func (r *MessageRPC) SaveGroup(m *myrpc.MessageSaveGroupArgs, ret *int) error {
	return nil
}

// GetPublish rpc interface get publish message.
func (r *MessageRPC) GetGroup(m *myrpc.MessageGetGroupArgs, rw *myrpc.MessageGetResp) error {
	return nil
}

// DelPublish rpc interface delete publish message.
func (r *MessageRPC) DelGroup(key string, ret *int) error {
	return nil
}
*/

// Server Ping interface
func (r *MessageRPC) Ping(p int, ret *int) error {
	glog.V(2).Info("ping ok")
	return nil
}
