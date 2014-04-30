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
	"errors"
	"github.com/Terry-Mao/gopush-cluster/hash"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/golang/glog"
	"net"
	"net/rpc"
)

var (
	ErrMigrate = errors.New("migrate nodes don't include self")
)

// StartRPC start rpc listen.
func StartRPC() {
	c := &CometRPC{}
	rpc.Register(c)
	for _, bind := range Conf.RPCBind {
		glog.Infof("start listen rpc addr: \"%s\"", bind)
		go rpcListen(bind)
	}
}

func rpcListen(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	// if process exit, then close the rpc bind
	defer func() {
		glog.Infof("rpc addr: \"%s\" close", bind)
		if err := l.Close(); err != nil {
			glog.Errorf("listener.Close() error(%v)", err)
		}
	}()
	rpc.Accept(l)
}

// Channel RPC
type CometRPC struct {
}

// New expored a method for creating new channel.
func (c *CometRPC) New(args *myrpc.CometNewArgs, ret *int) error {
	if args == nil || args.Key == "" {
		return myrpc.ErrParam
	}
	// create a new channel for the user
	ch, err := UserChannel.New(args.Key)
	if err != nil {
		glog.Errorf("UserChannel.New(\"%s\") error(%v)", args.Key, err)
		return err
	}
	if err = ch.AddToken(args.Key, args.Token); err != nil {
		glog.Errorf("ch.AddToken(\"%s\", \"%s\") error(%v)", args.Key, args.Token)
		return err
	}
	return nil
}

// Close expored a method for closing new channel.
func (c *CometRPC) Close(key string, ret *int) error {
	if key == "" {
		return myrpc.ErrParam
	}
	// close the channle for the user
	ch, err := UserChannel.Delete(key)
	if err != nil {
		glog.Errorf("UserChannel.Delete(\"%s\") error(%v)", key, err)
		return err
	}
	// ignore channel close error, only log a warnning
	if err := ch.Close(); err != nil {
		glog.Errorf("ch.Close() error(%v)", err)
		return err
	}
	return nil
}

// PushPrivate expored a method for publishing a user private message for the channel
func (c *CometRPC) PushPrivate(args *myrpc.CometPushPrivateArgs, ret *int) error {
	if args == nil || args.Key == "" || args.Msg == nil {
		return myrpc.ErrParam
	}
	// get a user channel
	ch, err := UserChannel.New(args.Key)
	if err != nil {
		glog.Errorf("UserChannel.New(\"%s\") error(%v)", args.Key, err)
		return err
	}
	// use the channel push message
	m := &myrpc.Message{Msg: args.Msg}
	if err = ch.PushMsg(args.Key, m, args.Expire); err != nil {
		glog.Errorf("ch.PushMsg(\"%s\", \"%v\") error(%v)", args.Key, m, err)
		return err
	}
	return nil
}

// Publish expored a method for publishing a message for the channel
func (c *CometRPC) Migrate(args *myrpc.CometMigrateArgs, ret *int) error {
	if len(args.Nodes) == 0 {
		return myrpc.ErrParam
	}
	// find current node exists in new nodes
	has := false
	for _, str := range args.Nodes {
		if str == Conf.ZookeeperCometNode {
			has = true
		}
	}
	if !has {
		glog.Error("make sure your migrate nodes right, there is no %s in nodes, this will cause all the node hit miss", Conf.ZookeeperCometNode)
		return ErrMigrate
	}
	// init ketama
	ketama := hash.NewKetama2(args.Nodes, args.Vnode)
	channels := []Channel{}
	keys := []string{}
	// get all the channel lock
	for i, c := range UserChannel.Channels {
		c.Lock()
		for k, v := range c.Data {
			hn := ketama.Node(k)
			if hn != Conf.ZookeeperCometNode {
				channels = append(channels, v)
				keys = append(keys, k)
				glog.V(1).Infof("migrate key:\"%s\" hit node:\"%s\"", k, hn)
			}
		}
		for _, k := range keys {
			delete(c.Data, k)
			glog.Infof("migrate delete channel key \"%s\"", k)
		}
		c.Unlock()
		glog.Infof("migrate channel bucket:%d finished", i)
	}
	// close all the migrate channels
	glog.Info("close all the migrate channels")
	for _, channel := range channels {
		if err := channel.Close(); err != nil {
			glog.Errorf("channel.Close() error(%v)", err)
			continue
		}
	}
	glog.Info("close all the migrate channels finished")
	return nil
}

func (c *CometRPC) Ping(args int, ret *int) error {
	glog.V(2).Info("ping ok")
	return nil
}
