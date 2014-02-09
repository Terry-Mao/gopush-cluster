package main

import (
	"github.com/Terry-Mao/gopush-cluster/hash"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"net/rpc"
	"time"
)

const (
	// internal failed
	retInternalErr = 65535
	// param error
	retParamErr = 65534
	// ok
	retOK = 0
	// create channel failed
	retCreateChannelErr = 1
	// add channel failed
	retAddChannleErr = 2
	// get channel failed
	retGetChannelErr = 3
	// message push failed
	retPushMsgErr = 4
	// migrate failed
	retMigrateErr = 5
	// rpc failed
	retRPCErr = 6
	// add token
	retAddTokenErr = 7
)

var (
	// rpc
	MsgRPC *rpc.Client
)

// InitMessageRPC init the message rpc connection.
func InitMessageRPC() {
	go func() {
		failed := false
		// if process exit, then close message rpc
		defer func() {
			if MsgRPC != nil {
				Log.Error("message rpc close")
				if err := MsgRPC.Close(); err != nil {
					Log.Error("MsgRPC.Close() error(%v)", err)
				}
			}
		}()
		for {
			if !failed && MsgRPC != nil {
				reply := 0
				if err := MsgRPC.Call("MessageRPC.Ping", 0, &reply); err != nil {
					Log.Error("rpc.Call(\"MessageRPC.Ping\") error(%v)", err)
					failed = true
				} else {
					// every one second send a heartbeat ping
					failed = false
					Log.Debug("rpc ping ok")
					time.Sleep(Conf.RPCPing)
					continue
				}
			}
			// reconnect(init) message rpc
			if rpcTmp, err := rpc.Dial("tcp", Conf.RPCMessageAddr); err != nil {
				Log.Error("rpc.Dial(\"tcp\", %s) error(%s), reconnect retry after %d second", Conf.RPCMessageAddr, err, int64(Conf.RPCRetry)/Second)
				time.Sleep(Conf.RPCRetry)
				continue
			} else {
				MsgRPC = rpcTmp
				failed = false
				Log.Info("rpc client reconnect \"%s\" ok", Conf.RPCMessageAddr)
			}
		}
	}()
}

// StartRPC start rpc listen.
func StartRPC() {
	c := &ChannelRPC{}
	rpc.Register(c)
	for _, bind := range Conf.RPCBind {
		Log.Info("start listen rpc addr:\"%s\"", bind)
		go rpcListen(bind)
	}
}

func rpcListen(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		Log.Error("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	// if process exit, then close the rpc bind
	defer func() {
		Log.Info("rpc addr: \"%s\" close", bind)
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() error(%v)", err)
		}
	}()
	rpc.Accept(l)
}

// Channel RPC
type ChannelRPC struct {
}

// New expored a method for creating new channel
func (c *ChannelRPC) New(args *myrpc.ChannelNewArgs, ret *int) error {
	if args == nil || args.Key == "" {
		*ret = retParamErr
		return nil
	}
	// create a new channel for the user
	ch, err := UserChannel.New(args.Key)
	if err != nil {
		*ret = retCreateChannelErr
		return nil
	}
	if err = ch.AddToken(args.Key, args.Token); err != nil {
		*ret = retAddTokenErr
		return nil
	}
	*ret = retOK
	return nil
}

// Close expored a method for closing new channel
func (c *ChannelRPC) Close(key *string, ret *int) error {
	if *key == "" {
		*ret = retParamErr
		return nil
	}
	// close the channle for the user
	ch, err := UserChannel.Delete(*key)
	if err != nil {
		*ret = retGetChannelErr
		return nil
	}
	// ignore channel close error, only log a warnning
	ch.Close()
	*ret = retOK
	return nil
}

// PushPrivate expored a method for publishing a user private message for the channel
func (c *ChannelRPC) PushPrivate(args *myrpc.ChannelPushPrivateArgs, ret *int) error {
	if args == nil || args.Key == "" || args.Msg == "" {
		*ret = retParamErr
		return nil
	}
	expire := args.Expire
	// get a user channel
	ch, err := UserChannel.New(args.Key)
	if err != nil {
		*ret = retGetChannelErr
		return nil
	}
	// use the channel push message
	if err = ch.PushMsg(args.Key, &Message{Msg: args.Msg, Expire: time.Now().UnixNano() + expire*Second, GroupID: args.GroupID}); err != nil {
		*ret = retPushMsgErr
		MsgStat.IncrFailed(1)
		return nil
	}
	*ret = retOK
	MsgStat.IncrSucceed(1)
	return nil
}

// PushPublic expored a method for publishing a public message for the channel
func (c *ChannelRPC) PushPublic(args *myrpc.ChannelPushPublicArgs, ret *int) error {
	succeed := uint64(0)
	failed := uint64(0)
	// get all the channel lock
	m := &Message{Msg: args.Msg, MsgID: args.MsgID, GroupID: myrpc.PublicGroupID}
	for _, c := range UserChannel.Channels {
		c.Lock()
		for k, v := range c.Data {
			if err := v.PushMsg(k, m); err != nil {
				// *ret = retPushMsgErr
				failed++
				continue
			}
			succeed++
		}
		c.Unlock()
	}
	MsgStat.IncrFailed(failed)
	MsgStat.IncrSucceed(succeed)
	*ret = retOK
	return nil
}

// Publish expored a method for publishing a message for the channel
func (c *ChannelRPC) Migrate(args *myrpc.ChannelMigrateArgs, ret *int) error {
	if len(args.Nodes) == 0 {
		*ret = retParamErr
		return nil
	}
	// find current node exists in new nodes
	has := false
	for _, str := range args.Nodes {
		if str == Conf.ZookeeperNode {
			has = true
		}
	}
	if !has {
		Log.Crit("make sure your migrate nodes right, there is no %s in nodes, this will cause all the node hit miss", Conf.ZookeeperNode)
		*ret = retMigrateErr
		return nil
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
			if hn != Conf.ZookeeperNode {
				channels = append(channels, v)
				keys = append(keys, k)
				Log.Debug("migrate key:\"%s\" hit node:\"%s\"", k, hn)
			}
		}
		for _, k := range keys {
			delete(c.Data, k)
			Log.Info("migrate delete channel key \"%s\"", k)
		}
		c.Unlock()
		Log.Info("migrate channel bucket:%d finished", i)
	}
	// close all the migrate channels
	Log.Info("close all the migrate channels")
	for _, channel := range channels {
		if err := channel.Close(); err != nil {
			Log.Error("channel.Close() error(%v)", err)
			continue
		}
	}
	*ret = retOK
	Log.Info("close all the migrate channels finished")
	return nil
}
