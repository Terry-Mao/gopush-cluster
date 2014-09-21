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
	log "code.google.com/p/log4go"
	"errors"
	"fmt"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"net/rpc"
	"strings"
	"sync"
)

var (
	ErrMigrate = errors.New("migrate nodes don't include self")
)

// StartRPC start rpc listen.
func StartRPC() error {
	c := &CometRPC{}
	rpc.Register(c)
	for _, bind := range Conf.RPCBind {
		addrs := strings.Split(bind, "-")
		if len(addrs) != 2 {
			return fmt.Errorf("config rpc.bind:\"%s\" format error", bind)
		}
		log.Info("start listen rpc addr: \"%s\"", bind)
		go rpcListen(addrs[1])
	}

	return nil
}

func rpcListen(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		log.Error("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	// if process exit, then close the rpc bind
	defer func() {
		log.Info("rpc addr: \"%s\" close", bind)
		if err := l.Close(); err != nil {
			log.Error("listener.Close() error(%v)", err)
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
		log.Error("UserChannel.New(\"%s\") error(%v)", args.Key, err)
		return err
	}
	if err = ch.AddToken(args.Key, args.Token); err != nil {
		log.Error("ch.AddToken(\"%s\", \"%s\") error(%v)", args.Key, args.Token)
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
		log.Error("UserChannel.Delete(\"%s\") error(%v)", key, err)
		return err
	}
	// ignore channel close error, only log a warnning
	if err := ch.Close(); err != nil {
		log.Error("ch.Close() error(%v)", err)
		return err
	}
	return nil
}

// PushPrivate expored a method for publishing a user private message for the channel.
// if it`s going failed then it`ll return an error
func (c *CometRPC) PushPrivate(args *myrpc.CometPushPrivateArgs, ret *int) error {
	if args == nil || args.Key == "" || args.Msg == nil {
		return myrpc.ErrParam
	}
	// get a user channel
	ch, err := UserChannel.New(args.Key)
	if err != nil {
		log.Error("UserChannel.New(\"%s\") error(%v)", args.Key, err)
		return err
	}
	// use the channel push message
	m := &myrpc.Message{Msg: args.Msg}
	if err = ch.PushMsg(args.Key, m, args.Expire); err != nil {
		log.Error("ch.PushMsg(\"%s\", \"%v\") error(%v)", args.Key, m, err)
		return err
	}
	return nil
}

// PushMultiplePrivate expored a method for publishing a user multiple private message for the channel.
// because of it`s going asynchronously in this method, so it won`t return an error to caller.
func (c *CometRPC) PushPrivates(args *myrpc.CometPushPrivatesArgs, rw *myrpc.CometPushPrivatesResp) error {
	if args == nil || args.Msg == nil {
		return myrpc.ErrParam
	}
	bucketMap := make(map[*ChannelBucket]map[string]Channel, Conf.ChannelBucket)
	for _, key := range args.Keys {
		// get channel
		ch, err := UserChannel.New(key)
		if err != nil {
			log.Error("UserChannel.New(\"%s\") error(%v)", key, err)
			// log failed keys.
			rw.FKeys = append(rw.FKeys, key)
		}
		// get bucket index
		bp := UserChannel.Bucket(key)
		if bucket, ok := bucketMap[bp]; !ok {
			bucketMap[bp] = map[string]Channel{key: ch}
		} else {
			bucket[key] = ch
		}
	}
	// every bucket start a goroutine, return till all bucket gorouint finish
	//lock := &sync.Mutex{}
	msg := &myrpc.Message{Msg: args.Msg}
	wg := sync.WaitGroup{}
	wg.Add(len(bucketMap))
	for tb, tm := range bucketMap {
		go func(b *ChannelBucket, m map[string]Channel) {
			b.Lock()
			// TODO rpc push msgs
			// lock log failed keys.
			//lock.Lock()
			//rw.FKeys = append(rw.FKeys, key)
			//lock.Unlock()
			// comet push msgs
			for key, ch := range m {
				if err := ch.WriteMsg(key, msg); err != nil {
					log.Error("ch.WriteMsg(\"%s\", \"%s\") error(%v)", key, string(msg.Msg), err)
					continue
				}
			}
			b.Unlock()
			wg.Done()
		}(tb, tm)
	}
	wg.Wait()
	return nil
}

// Migrate update the inner hashring and node info.
func (c *CometRPC) Migrate(args *myrpc.CometMigrateArgs, ret *int) error {
	UserChannel.Migrate(args.Nodes)
	return nil
}

// Ping check health.
func (c *CometRPC) Ping(args int, ret *int) error {
	log.Debug("ping ok")
	return nil
}
