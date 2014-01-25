package main

import (
	"github.com/Terry-Mao/gopush-cluster/hash"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"net/rpc"
	"os"
	"strings"
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

//
func InitMessageRPC() error {
	var err error
	MsgRPC, err = rpc.Dial("tcp", Conf.RPCMessageAddr)
	if err != nil {
		Log.Error("rpc.Dial(\"tcp\", %s) failed (%s)", Conf.RPCMessageAddr, err.Error())
		return err
	}
	defer func() {
		if err := MsgRPC.Close(); err != nil {
			Log.Error("rpc.Close() failed (%s)", err.Error())
		}
	}()
	// rpc Ping
	go func() {
		for {
			reply := 0
			if err := MsgRPC.Call("MessageRPC.Ping", 0, &reply); err != nil {
				Log.Error("rpc.Call(\"MessageRPC.Ping\") failed (%s)", err.Error())
				rpcTmp, err := rpc.Dial("tcp", Conf.RPCMessageAddr)
				if err != nil {
					Log.Error("rpc.Dial(\"tcp\", %s) failed (%s)", Conf.RPCMessageAddr, err.Error())
					time.Sleep(Conf.RPCRetry)
					Log.Warn("rpc reconnect \"%s\" after %d second", Conf.RPCMessageAddr, Conf.RPCRetry)
					continue
				} else {
					Log.Info("rpc client reconnect \"%s\" ok", Conf.RPCMessageAddr)
					MsgRPC = rpcTmp
				}
			}
			// every one second send a heartbeat ping
			Log.Debug("rpc ping ok")
			time.Sleep(Conf.RPCPing)
		}
	}()
	return nil
}

// StartRPC start rpc listen.
func StartRPC() {
	c := &ChannelRPC{}
	rpc.Register(c)
	for _, bind := range strings.Split(Conf.AdminBind, ",") {
		Log.Info("start listen rpc addr:\"%s\"", bind)
		go rpcListen(bind)
	}
}

func rpcListen(bind string) {
	l, err := net.Listen("tcp", bind)
	if err != nil {
		Log.Error("net.Listen(\"tcp\", \"%s\") failed (%s)", bind, err.Error())
		os.Exit(-1)
	}
	defer func() {
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() failed (%s)", err.Error())
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

// Publish expored a method for publishing a message for the channel
func (c *ChannelRPC) Publish(args *myrpc.ChannelPublishArgs, ret *int) error {
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
	if err = ch.PushMsg(&Message{Msg: args.Msg, Expire: time.Now().UnixNano() + expire*Second, MsgID: args.MsgID, GroupID: args.GroupID}, args.Key); err != nil {
		*ret = retPushMsgErr
		MsgStat.IncrFailed()
		return nil
	}
	*ret = retOK
	MsgStat.IncrSucceed()
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
			Log.Error("channel.Close() failed (%s)", err.Error())
			continue
		}
	}
	*ret = retOK
	Log.Info("close all the migrate channels finished")
	return nil
}
