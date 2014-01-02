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
)

var (
	// rpc
	RPCCli *rpc.Client
)

// StartRPC start accept rpc call
func StartRPC() error {
	var err error

	if Conf.ChannelType == OuterChannelType {
		RPCCli, err = rpc.Dial("tcp", Conf.RPCAddr)
		if err != nil {
			Log.Error("rpc.Dial(\"tcp\", %s) failed (%s)", Conf.RPCAddr, err.Error())
			return err
		}

		defer func() {
			if err := RPCCli.Close(); err != nil {
				Log.Error("rpc.Close() failed (%s)", err.Error())
			}
		}()

		// rpc Ping
		go func() {
			for {
				reply := 0
				if err := RPCCli.Call("WebRPC.Ping", 0, &reply); err != nil {
					Log.Error("rpc.Call(\"WebRPC.Ping\") failed (%s)", err.Error())
					rpcTmp, err := rpc.Dial("tcp", Conf.RPCAddr)
					if err != nil {
						Log.Error("rpc.Dial(\"tcp\", %s) failed (%s)", Conf.RPCAddr, err.Error())
						time.Sleep(time.Duration(Conf.RPCRetrySec) * time.Second)
						Log.Warn("rpc reconnect \"%s\" after %d second", Conf.RPCAddr, Conf.RPCRetrySec)
                        continue
					} else {
						Log.Info("rpc client reconnect \"%s\" ok", Conf.RPCAddr)
						RPCCli = rpcTmp
					}
				}

				// every one second send a heartbeat ping
				Log.Debug("rpc ping ok")
				time.Sleep(time.Duration(Conf.RPCHeartbeatSec) * time.Second)
			}
		}()
	}

	c := &ChannelRPC{}
	rpc.Register(c)
	l, err := net.Listen("tcp", Conf.AdminAddr)
	if err != nil {
		Log.Error("net.Listen(\"tcp\", \"%s\") failed (%s)", Conf.AdminAddr, err.Error())
		return err
	}

	defer func() {
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() failed (%s)", err.Error())
		}
	}()

	Log.Info("start listen rpc addr:%s", Conf.AdminAddr)
	rpc.Accept(l)
	return nil
}

// Channel RPC
type ChannelRPC struct {
}

// New expored a method for creating new channel
func (c *ChannelRPC) New(key *string, ret *int) error {
	if *key == "" {
		Log.Warn("ChannelRPC New param error")
		*ret = retParamErr
		return nil
	}

	// create a new channel for the user
	Log.Info("user_key:\"%s\" add channel", *key)
	_, err := UserChannel.New(*key)
	if err != nil {
		Log.Error("user_key:\"%s\" can't create channle", *key)
		*ret = retCreateChannelErr

		return nil
	}

	*ret = retOK
	return nil
}

// Close expored a method for closing new channel
func (c *ChannelRPC) Close(key *string, ret *int) error {
	if *key == "" {
		Log.Warn("ChannelRPC Close param error")
		*ret = retParamErr
		return nil
	}

	// close the channle for the user
	Log.Info("user_key:\"%s\" close channel", *key)
	ch, err := UserChannel.Get(*key)
	if err != nil {
		Log.Error("user_key:\"%s\" can't get channle", *key)
		*ret = retGetChannelErr

		return nil
	}

	// ignore channel close error, only log a warnning
	if err = ch.Close(); err != nil {
		Log.Warn("user_key:\"%s\" can't close channel", *key)
	}

	*ret = retOK
	return nil
}

// Publish expored a method for publishing a message for the channel
func (c *ChannelRPC) Publish(m *myrpc.ChannelPublishArgs, ret *int) error {
	if m == nil || m.Key == "" || m.Msg == "" {
		Log.Warn("ChannelRPC Publish param error")
		*ret = retParamErr
		return nil
	}

	expire := m.Expire
	if expire <= 0 {
		expire = Conf.MessageExpireSec
	}

	// get a user channel
	ch, err := UserChannel.New(m.Key)
	if err != nil {
		Log.Warn("user_key:\"%s\" can't get a channel (%s)", m.Key, err.Error())
		*ret = retGetChannelErr
		return nil
	}

	// use the channel push message
	if err = ch.PushMsg(&Message{Msg: m.Msg, Expire: time.Now().UnixNano() + expire*Second, MsgID: m.MsgID}, m.Key); err != nil {
		Log.Error("user_key:\"%s\" push message failed (%s)", m.Key, err.Error())
		*ret = retPushMsgErr
		MsgStat.IncrFailed()
		return nil
	}

	MsgStat.IncrSucceed()
	*ret = retOK
	return nil
}

// Publish expored a method for publishing a message for the channel
func (c *ChannelRPC) Migrate(m *myrpc.ChannelMigrateArgs, ret *int) error {
	if len(m.Nodes) == 0 {
		Log.Warn("ChannelRPC Migrate param error")
		*ret = retParamErr
		return nil
	}

	// find current node exists in new nodes
	has := false
	for _, str := range m.Nodes {
		if str == Conf.Node {
			has = true
		}
	}

	if !has {
		Log.Crit("make sure your migrate nodes right, there is no %s in nodes, this will cause all the node hit miss", Conf.Node)
		*ret = retMigrateErr
		return nil
	}

	// init ketama
	ketama := hash.NewKetama2(m.Nodes, m.Vnode)
	channels := []Channel{}
	// get all the channel lock
	for i, c := range UserChannel.Channels {
		Log.Info("migrate channel bucket:%d", i)
		c.Lock()
		for k, v := range c.Data {
			hn := ketama.Node(k)
			if hn != Conf.Node {
				channels = append(channels, v)
				Log.Debug("migrate key:\"%s\" hit node:\"%s\"", k, hn)
			}
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

	Log.Info("close all the migrate channels finished")
	*ret = retOK
	return nil
}
