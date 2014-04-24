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

// github.com/samuel/go-zookeeper
// Copyright (c) 2013, Samuel Stauffer <samuel@descolada.com>
// All rights reserved.

package main

import (
	"fmt"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"path"
	"strings"
	"time"
)

const (
	messageService = "MessageRPC"
	// node event
	eventNodeAdd    = 1
	eventNodeDel    = 2
	eventNodeUpdate = 3

	// wait node
	waitNodeDelay       = 3
	waitNodeDelaySecond = waitNodeDelay * time.Second
)

var (
	MessageRPC *myrpc.RandLB
)

type NodeEvent struct {
	// addr:port
	Key string
	// event type
	Event int
}

func init() {
	MessageRPC, _ = myrpc.NewRandLB(map[string]*rpc.Client{}, []string{}, messageService, 0, 0, false)
}

func InitZK() (*zk.Conn, error) {
	conn, err := myzk.Connect(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		glog.Errorf("zk.Connect() error(%v)", err)
		return nil, err
	}
	fpath := path.Join(Conf.ZookeeperCometPath, Conf.ZookeeperCometNode)
	if err = myzk.Create(conn, fpath); err != nil {
		glog.Errorf("zk.Create() error(%v)", err)
		return conn, err
	}
	// tcp, websocket and rpc bind address store in the zk
	data := ""
	for _, addr := range Conf.TCPBind {
		data += fmt.Sprintf("tcp://%s,", addr)
	}
	for _, addr := range Conf.WebsocketBind {
		data += fmt.Sprintf("ws://%s,", addr)
	}
	for _, addr := range Conf.RPCBind {
		data += fmt.Sprintf("rpc://%s,", addr)
	}
	if err = myzk.RegisterTemp(conn, fpath, strings.TrimRight(data, ",")); err != nil {
		glog.Errorf("zk.RegisterTemp() error(%v)", err)
		return conn, err
	}
	// watch message path
	ch := make(chan *NodeEvent, 1024)
	go handleNodeEvent(conn, Conf.ZookeeperMessagePath, ch)
	go watchRoot(conn, Conf.ZookeeperMessagePath, ch)
	return conn, nil
}

// watchRoot watch the message root path.
func watchRoot(conn *zk.Conn, fpath string, ch chan *NodeEvent) error {
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err == myzk.ErrNodeNotExist {
			glog.Warningf("zk don't have node \"%s\", retry in %d second", fpath, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err == myzk.ErrNoChild {
			glog.Warningf("zk don't have any children in \"%s\", retry in %d second", fpath, waitNodeDelay)
			// all child died, kick all the nodes
			for node, _ := range MessageRPC.Clients {
				glog.V(1).Infof("node: \"%s\" send del node event", node)
				ch <- &NodeEvent{Event: eventNodeDel, Key: node}
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
			addrs := strings.Split(string(data), ",")
			for _, addr := range addrs {
				// if not exists in old map then trigger a add event
				if _, ok := MessageRPC.Clients[addr]; !ok {
					ch <- &NodeEvent{Event: eventNodeAdd, Key: addr}
				}
				nodesMap[addr] = true
			}
		}
		// handle delete nodes
		for node, _ := range MessageRPC.Clients {
			if _, ok := nodesMap[node]; !ok {
				ch <- &NodeEvent{Event: eventNodeDel, Key: node}
			}
		}
		// blocking wait node changed
		event := <-watch
		glog.Infof("zk path: \"%s\" receive a event %v", fpath, event)
	}
}

// handleNodeEvent add and remove MessageRPC.Clients, copy the src map to a new map then replace the variable.
func handleNodeEvent(conn *zk.Conn, path string, ch chan *NodeEvent) {
	for {
		ev := <-ch
		// copy map from src
		tmpMessageRPCMap := make(map[string]*rpc.Client, len(MessageRPC.Clients))
		for k, v := range MessageRPC.Clients {
			tmpMessageRPCMap[k] = v
		}
		// handle event
		if ev.Event == eventNodeAdd {
			glog.Infof("add message rpc node: \"%s\"", ev.Key)
			// if exist old node info, reuse
			if client, ok := MessageRPC.Clients[ev.Key]; ok && client != nil {
				tmpMessageRPCMap[ev.Key] = client
				glog.Infof("reuse message rpc node: \"%s\"", ev.Key)
			} else {
				rpcTmp, err := rpc.Dial("tcp", ev.Key)
				if err != nil {
					glog.Errorf("rpc.Dial(\"tcp\", \"%s\") error(%v)", ev.Key, err)
					glog.Warningf("discard message rpc node: \"%s\", connect failed", ev.Key)
					continue
				}
				tmpMessageRPCMap[ev.Key] = rpcTmp
			}
		} else if ev.Event == eventNodeDel {
			glog.Infof("del message rpc node: \"%s\"", ev.Key)
			delete(tmpMessageRPCMap, ev.Key)
			// if exist old node info, close
			if client, ok := MessageRPC.Clients[ev.Key]; ok && client != nil {
				client.Close()
				glog.Infof("close message rpc node: \"%s\"", ev.Key)
			}
		} else {
			glog.Errorf("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		addrs := make([]string, 0, len(tmpMessageRPCMap))
		for addr, _ := range tmpMessageRPCMap {
			addrs = append(addrs, addr)
		}
		tmpMessageRPC, err := myrpc.NewRandLB(tmpMessageRPCMap, addrs, messageService, Conf.RPCRetry, Conf.RPCPing, true)
		if err != nil {
			glog.Errorf("rpc.NewRandLR() error(%v)", err)
			panic(err)
		}
		// atomic update
		MessageRPC.Stop()
		MessageRPC = tmpMessageRPC
		glog.V(1).Infof("MessageRPC.Client length: %d", len(MessageRPC.Clients))
	}
}
