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
	log "code.google.com/p/log4go"
	"encoding/json"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"path"
	"time"
)

const (
	// group id
	PrivateGroupId = 0
	PublicGroupId  = 1
	// message rpc service
	MessageService             = "MessageRPC"
	MessageServiceGetPrivate   = "MessageRPC.GetPrivate"
	MessageServiceSavePrivate  = "MessageRPC.SavePrivate"
	MessageServiceSavePrivates = "MessageRPC.SavePrivates"
	MessageServiceDelPrivate   = "MessageRPC.DelPrivate"
)

var (
	MessageRPC *RandLB
)

func init() {
	MessageRPC, _ = NewRandLB(map[string]*WeightRpc{}, MessageService, 0, 0, 255, false)
}

type MessageNodeEvent struct {
	Key *WeightRpc
	// event type
	Event int
}

// Message node info
type MessageNodeInfo struct {
	Rpc    []string `json:"rpc"`
	Weight int      `json:"weight"`
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
		log.Error("json.Marshal(%v) error(%v)", m, err)
		return nil, err
	}
	return byteJson, nil
}

// OldBytes get a message reply bytes(Compatible), TODO remove it.
func (m *Message) OldBytes() ([]byte, error) {
	om := &OldMessage{Msg: string(m.Msg), MsgId: m.MsgId, GroupId: m.GroupId}
	byteJson, err := json.Marshal(om)
	if err != nil {
		log.Error("json.Marshal(%v) error(%v)", om, err)
		return nil, err
	}
	return byteJson, nil
}

// Message SavePrivate args
type MessageSavePrivateArgs struct {
	Key    string          // subscriber key
	Msg    json.RawMessage // message content
	MsgId  int64           // message id
	Expire uint            // message expire second
}

// Message SavePrivates args
type MessageSavePrivatesArgs struct {
	Keys   []string        // subscriber keys
	Msg    json.RawMessage // message content
	MsgId  int64           // message id
	Expire uint            // message expire second
}

// Message SavePrivates response
type MessageSavePrivatesResp struct {
	FKeys []string // failed key
}

// Message SavePublish args
type MessageSavePublishArgs struct {
	MsgID  int64  // message id
	Msg    string // message content
	Expire int64  // message expire second
}

// Message Get args
type MessageGetPrivateArgs struct {
	MsgId int64  // message id
	Key   string // subscriber key
}

// Message Get Response
type MessageGetResp struct {
	Msgs []*Message // messages
}

// watchMessageRoot watch the message root path.
func watchMessageRoot(conn *zk.Conn, fpath string, ch chan *MessageNodeEvent) error {
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err == myzk.ErrNodeNotExist {
			log.Warn("zk don't have node \"%s\", retry in %d second", fpath, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err == myzk.ErrNoChild {
			log.Warn("zk don't have any children in \"%s\", retry in %d second", fpath, waitNodeDelay)
			// all child died, kick all the nodes
			for _, client := range MessageRPC.Clients {
				log.Debug("node: \"%s\" send del node event", client.Addr)
				ch <- &MessageNodeEvent{Event: eventNodeDel, Key: &WeightRpc{Addr: client.Addr, Weight: client.Weight}}
			}
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err != nil {
			log.Error("getNodes error(%v), retry in %d second", err, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		}
		nodesMap := map[string]bool{}
		// handle new add nodes
		for _, node := range nodes {
			data, _, err := conn.Get(path.Join(fpath, node))
			if err != nil {
				log.Error("zk.Get(\"%s\") error(%v)", path.Join(fpath, node), err)
				continue
			}
			// parse message node info
			nodeInfo := &MessageNodeInfo{}
			if err := json.Unmarshal(data, nodeInfo); err != nil {
				log.Error("json.Unmarshal(\"%s\", nodeInfo) error(%v)", string(data), err)
				continue
			}
			for _, addr := range nodeInfo.Rpc {
				// if not exists in old map then trigger a add event
				if _, ok := MessageRPC.Clients[addr]; !ok {
					ch <- &MessageNodeEvent{Event: eventNodeAdd, Key: &WeightRpc{Addr: addr, Weight: nodeInfo.Weight}}
				}
				nodesMap[addr] = true
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
		log.Info("zk path: \"%s\" receive a event %v", fpath, event)
	}
}

// handleNodeEvent add and remove MessageRPC.Clients, copy the src map to a new map then replace the variable.
func handleMessageNodeEvent(conn *zk.Conn, retry, ping time.Duration, vnode int, ch chan *MessageNodeEvent) {
	for {
		ev := <-ch
		// copy map from src
		tmpMessageRPCMap := make(map[string]*WeightRpc, len(MessageRPC.Clients))
		for k, v := range MessageRPC.Clients {
			tmpMessageRPCMap[k] = v
			// reuse rpc connection
			v.Client = nil
		}
		// handle event
		if ev.Event == eventNodeAdd {
			log.Info("add message rpc node: \"%s\"", ev.Key.Addr)
			rpcTmp, err := rpc.Dial("tcp", ev.Key.Addr)
			if err != nil {
				log.Error("rpc.Dial(\"tcp\", \"%s\") error(%v)", ev.Key, err)
				log.Warn("discard message rpc node: \"%s\", connect failed", ev.Key)
				continue
			}
			ev.Key.Client = rpcTmp
			tmpMessageRPCMap[ev.Key.Addr] = ev.Key
		} else if ev.Event == eventNodeDel {
			log.Info("del message rpc node: \"%s\"", ev.Key.Addr)
			delete(tmpMessageRPCMap, ev.Key.Addr)
		} else {
			log.Error("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		tmpMessageRPC, err := NewRandLB(tmpMessageRPCMap, MessageService, retry, ping, vnode, true)
		if err != nil {
			log.Error("NewRandLR() error(%v)", err)
			panic(err)
		}
		oldMessageRPC := MessageRPC
		// atomic update
		MessageRPC = tmpMessageRPC
		// release resource
		oldMessageRPC.Destroy()
		log.Debug("MessageRPC.Client length: %d", len(MessageRPC.Clients))
	}
}

// InitMessage init a rand lb rpc for message module.
func InitMessage(conn *zk.Conn, fpath string, retry, ping time.Duration, vnode int) {
	// watch message path
	ch := make(chan *MessageNodeEvent, 1024)
	go handleMessageNodeEvent(conn, retry, ping, vnode, ch)
	go watchMessageRoot(conn, fpath, ch)
}
