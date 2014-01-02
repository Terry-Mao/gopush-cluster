package main

import (
	"encoding/json"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"net"
	"net/rpc"
)

// RPC For receive offline messages
type MessageRPC struct {
}

// StartRPC start accept rpc call
func StartRPC() error {
	r := &MessageRPC{}
	rpc.Register(r)
	l, err := net.Listen("tcp", Conf.InternalAddr)
	if err != nil {
		Log.Error("net.Listen(\"tcp\", \"%s\") failed (%v)", Conf.InternalAddr, err)
		return err
	}

	defer func() {
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() failed (%v)", err)
		}
	}()

	Log.Info("start listen admin addr:%s", Conf.InternalAddr)
	rpc.Accept(l)
	return nil
}

func (r *MessageRPC) Save(m *myrpc.MessageSaveArgs, ret *int) error {
	if m == nil || m.Key == "" || m.Msg == "" {
		*ret = ParamErr
		return nil
	}

	// Json.Marshal and save the message
	recordMsg := Message{Msg: m.Msg, Expire: m.Expire, MsgID: m.MsgID}
	message, _ := json.Marshal(recordMsg)
	if err := SaveMessage(m.Key, string(message), m.MsgID); err != nil {
		Log.Error("save message error (%v)", err)
		*ret = InternalErr
		return nil
	}

	*ret = OK
	return nil
}
