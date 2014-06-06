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

package rpc

import (
	"encoding/json"
	"fmt"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	// group id
	PrivateGroupId = 0
	PublicGroupId  = 1
	// message rpc service
	MessageService            = "MessageRPC"
	MessageServiceGetPrivate  = "MessageRPC.GetPrivate"
	MessageServiceSavePrivate = "MessageRPC.SavePrivate"
	MessageServiceDelPrivate  = "MessageRPC.DelPrivate"
)

type MessageNodeEvent struct {
	Key *RPCClient
	// event type
	Event int
}

// The Message struct
type Message struct {
	Msg     json.RawMessage `json:"msg"` // message content
	MsgId   int64           `json:"mid"` // message id
	GroupId uint            `json:"gid"` // group id
}

// The Old Message struct (Compatible), TODO remove it.
type OldMessage struct {
	Msg     string `json:"msg"` // Message
	MsgId   int64  `json:"mid"` // Message id
	GroupId uint   `json:"gid"` // Group id
}

// Bytes get a message reply bytes.
func (m *Message) Bytes() ([]byte, error) {
	byteJson, err := json.Marshal(m)
	if err != nil {
		glog.Errorf("json.Marshal(%v) error(%v)", m, err)
		return nil, err
	}
	return byteJson, nil
}

// OldBytes get a message reply bytes(Compatible), TODO remove it.
func (m *Message) OldBytes() ([]byte, error) {
	om := &OldMessage{Msg: string(m.Msg), MsgId: m.MsgId, GroupId: m.GroupId}
	byteJson, err := json.Marshal(om)
	if err != nil {
		glog.Errorf("json.Marshal(%v) error(%v)", om, err)
		return nil, err
	}
	return byteJson, nil
}

// Message SavePrivate Args
type MessageSavePrivateArgs struct {
	Key    string          // subscriber key
	Msg    json.RawMessage // message content
	MsgId  int64           // message id
	Expire uint            // message expire second
}

// Message SavePublish Args
type MessageSavePublishArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
}

// Message Get Args
type MessageGetPrivateArgs struct {
	MsgId int64  // message id
	Key   string // subscriber key
}

// Message Get Response
type MessageGetResp struct {
	Msgs []*Message // messages
}

var (
	MessageRPC *RandLB
)

func init() {
	MessageRPC, _ = NewRandLB(map[string]*RPCClient{}, MessageService, 0, 0, 255, false)
}

// watchMessageRoot watch the message root path.
func watchMessageRoot(conn *zk.Conn, fpath string, ch chan *MessageNodeEvent) error {
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err == myzk.ErrNodeNotExist {
			glog.Warningf("zk don't have node \"%s\", retry in %d second", fpath, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err == myzk.ErrNoChild {
			glog.Warningf("zk don't have any children in \"%s\", retry in %d second", fpath, waitNodeDelay)
			// all child died, kick all the nodes
			for _, client := range MessageRPC.Clients {
				glog.V(1).Infof("node: \"%s\" send del node event", client.Addr)
				ch <- &MessageNodeEvent{Event: eventNodeDel, Key: &RPCClient{Addr: client.Addr, Weight: client.Weight}}
			}
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err != nil {
			glog.Errorf("getNodes error(%v), retry in %d second", err, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		}
		nodesMap := map[string]bool{}
		// handle new add nodes
		for _, node := range nodes {
			data, _, err := conn.Get(path.Join(fpath, node))
			if err != nil {
				glog.Errorf("zk.Get(\"%s\") error(%v)", path.Join(fpath, node), err)
				continue
			}
			// may contains many addrs split by ,
			//addrs := strings.Split(string(data), ",")
			addrInfos, err := parseMessageAddr(string(data))
			if err != nil {
				glog.Errorf("parseMessageAddr(\"%s\") error(%v)", string(data), err)
				continue
			}
			for _, addInfo := range addrInfos {
				// if not exists in old map then trigger a add event
				if _, ok := MessageRPC.Clients[addInfo.Addr]; !ok {
					ch <- &MessageNodeEvent{Event: eventNodeAdd, Key: addInfo}
				}
				nodesMap[addInfo.Addr] = true
			}
		}
		// handle delete nodes
		for _, client := range MessageRPC.Clients {
			if _, ok := nodesMap[client.Addr]; !ok {
				ch <- &MessageNodeEvent{Event: eventNodeDel, Key: client}
			}
		}
		// blocking wait node changed
		event := <-watch
		glog.Infof("zk path: \"%s\" receive a event %v", fpath, event)
	}
}

// parseMessageAddr parse message listener addrs like:1-ip:port,2-ip:port
func parseMessageAddr(data string) (res []*RPCClient, err error) {
	AddrArr := strings.Split(data, ",") // eg 1-ip:port,2-ip:port
	for _, addr := range AddrArr {
		wArr := strings.Split(addr, "-") // eg: 1-ip:port
		if len(wArr) != 2 {
			err = fmt.Errorf("data:\"%s\" format error", data)
			return
		}
		var wAddr int
		wAddr, err = strconv.Atoi(wArr[0])
		if err != nil {
			err = fmt.Errorf("data:\"%s\" format error(%v)", data, err)
			return
		}
		res = append(res, &RPCClient{Addr: wArr[1], Weight: wAddr})
	}

	return
}

// handleNodeEvent add and remove MessageRPC.Clients, copy the src map to a new map then replace the variable.
func handleMessageNodeEvent(conn *zk.Conn, retry, ping time.Duration, vnode int, ch chan *MessageNodeEvent) {
	for {
		ev := <-ch
		// copy map from src
		tmpMessageRPCMap := make(map[string]*RPCClient, len(MessageRPC.Clients))
		for k, v := range MessageRPC.Clients {
			tmpMessageRPCMap[k] = v
		}
		// handle event
		if ev.Event == eventNodeAdd {
			glog.Infof("add message rpc node: \"%s\"", ev.Key.Addr)
			// if exist old node info, reuse
			if client, ok := MessageRPC.Clients[ev.Key.Addr]; ok && client != nil {
				tmpMessageRPCMap[ev.Key.Addr] = client
				glog.Infof("reuse message rpc node: \"%s\"", ev.Key.Addr)
			} else {
				rpcTmp, err := rpc.Dial("tcp", ev.Key.Addr)
				if err != nil {
					glog.Errorf("rpc.Dial(\"tcp\", \"%s\") error(%v)", ev.Key, err)
					glog.Warningf("discard message rpc node: \"%s\", connect failed", ev.Key)
					continue
				}
				ev.Key.Client = rpcTmp
				tmpMessageRPCMap[ev.Key.Addr] = ev.Key
			}
		} else if ev.Event == eventNodeDel {
			glog.Infof("del message rpc node: \"%s\"", ev.Key.Addr)
			delete(tmpMessageRPCMap, ev.Key.Addr)
			// if exist old node info, close
			if client, ok := MessageRPC.Clients[ev.Key.Addr]; ok && client != nil {
				client.Client.Close()
				glog.Infof("close message rpc node: \"%s\"", ev.Key.Addr)
			}
		} else {
			glog.Errorf("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		tmpMessageRPC, err := NewRandLB(tmpMessageRPCMap, MessageService, retry, ping, vnode, true)
		if err != nil {
			glog.Errorf("NewRandLR() error(%v)", err)
			panic(err)
		}
		// atomic update
		MessageRPC.Stop()
		MessageRPC = tmpMessageRPC
		glog.V(1).Infof("MessageRPC.Client length: %d", len(MessageRPC.Clients))
	}
}

// InitMessage init a rand lb rpc for message module.
func InitMessage(conn *zk.Conn, fpath string, retry, ping time.Duration, vnode int) {
	// watch message path
	ch := make(chan *MessageNodeEvent, 1024)
	go handleMessageNodeEvent(conn, retry, ping, vnode, ch)
	go watchMessageRoot(conn, fpath, ch)
}
