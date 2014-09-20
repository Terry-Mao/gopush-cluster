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
	"errors"
	"github.com/Terry-Mao/gopush-cluster/ketama"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"path"
	"sort"
	"strconv"
	"time"
)

const (
	cometService             = "CometRPC"
	CometServicePushPrivate  = "CometRPC.PushPrivate"
	CometServicePushPrivates = "CometRPC.PushPrivates"
)

var (
	// Store the first alive Comet service of every node
	// If there is no alive service under the node, the map`s value will be nil, but key is exist in map
	cometNodeInfoMap = make(map[string]*CometNodeInfo)
	// Ketama algorithm for check Comet node
	cometRing   *ketama.HashRing
	ErrCometRPC = errors.New("comet rpc call failed")
)

// CometNodeData stored in zookeeper
type CometNodeInfo struct {
	RpcAddr []string `json:"ws"`
	TcpAddr []string `json:"tcp"`
	WsAddr  []string `json:"rpc"`
	Weight  int      `json:"weight"`
	Rpc     *RandLB  `json:"-"`
}

type CometNodeEvent struct {
	// node name(node1, node2...)
	Key string
	// node info
	Value *CometNodeInfo
	// event type
	Event int
	// Migrate
	Migrate bool
}

// Channel Push Private Message Args
type CometPushPrivateArgs struct {
	Key    string          // subscriber key
	Msg    json.RawMessage // message content
	Expire uint            // message expire second
}

// Channel Push multi Private Message Args
type CometPushPrivatesArgs struct {
	Keys   []string        // subscriber keys
	Msg    json.RawMessage // message content
	Expire uint            // message expire second
}

// Channel Push multi Private Message response
type CometPushPrivatesResp struct {
	FKeys []string // subscriber keys
}

// Channel Push Public Message Args
type CometPushPublicArgs struct {
	MsgID int64  // message id
	Msg   string // message content
}

// Channel Migrate Args
type CometMigrateArgs struct {
	Nodes map[string]int // current comet nodes
	Vnode int            // ketama virtual node number
}

// Channel New Args
type CometNewArgs struct {
	Expire int64  // message expire second
	Token  string // auth token
	Key    string // subscriber key
}

// watchCometRoot watch the gopush root node for detecting the node add/del.
func watchCometRoot(conn *zk.Conn, fpath string, ch chan *CometNodeEvent) error {
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err == myzk.ErrNodeNotExist {
			log.Warn("zk don't have node \"%s\", retry in %d second", fpath, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err == myzk.ErrNoChild {
			log.Warn("zk don't have any children in \"%s\", retry in %d second", fpath, waitNodeDelay)
			for node, _ := range cometNodeInfoMap {
				ch <- &CometNodeEvent{Event: eventNodeDel, Key: node, Migrate: true}
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
			if _, ok := cometNodeInfoMap[node]; !ok {
				ch <- &CometNodeEvent{Event: eventNodeAdd, Key: node, Migrate: true}
			}
			nodesMap[node] = true
		}
		// handle delete nodes
		for node, _ := range cometNodeInfoMap {
			if _, ok := nodesMap[node]; !ok {
				ch <- &CometNodeEvent{Event: eventNodeDel, Key: node, Migrate: true}
			}
		}
		event := <-watch
		log.Info("zk path: \"%s\" receive a event %v", fpath, event)
	}
}

// handleCometNodeEvent add and remove CometNodeInfo, copy the src map to a new map then replace the variable.
func handleCometNodeEvent(conn *zk.Conn, fpath string, retry, ping time.Duration, vnode int, ch chan *CometNodeEvent) {
	for {
		ev := <-ch
		// copy map from src
		tmpMap := make(map[string]*CometNodeInfo, len(cometNodeInfoMap))
		for k, v := range cometNodeInfoMap {
			tmpMap[k] = v
		}
		// handle event
		if ev.Event == eventNodeAdd {
			log.Info("add node: \"%s\"", ev.Key)
			// get comet root node weight
			data, _, err := conn.Get(path.Join(fpath, ev.Key))
			if err != nil {
				log.Error("cannot get data from node:\"%s\"", path.Join(fpath, ev.Key))
				continue
			}
			weight, err := strconv.Atoi(string(data))
			if err != nil {
				log.Error("node:\"%s\" data:\"%s\" format error", path.Join(fpath, ev.Key), string(data))
				continue
			}
			log.Debug("get node:%s weight:%s", ev.Key, string(data))
			tmpMap[ev.Key] = &CometNodeInfo{Weight: weight}
			go watchCometNode(conn, ev.Key, fpath, weight, retry, ping, vnode, ch)
		} else if ev.Event == eventNodeDel {
			log.Info("del node: \"%s\"", ev.Key)
			delete(tmpMap, ev.Key)
		} else if ev.Event == eventNodeUpdate {
			log.Info("update node: \"%s\"", ev.Key)
			tmpMap[ev.Key] = ev.Value
		} else {
			log.Error("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		// if exist old node info, destroy
		if info, ok := cometNodeInfoMap[ev.Key]; ok {
			if info != nil && info.Rpc != nil {
				info.Rpc.Destroy()
			}
		}
		// update comet hash, cause node has changed
		tempRing := ketama.NewRing(vnode)
		for k, v := range tmpMap {
			log.Debug("AddNode node:%s weight:%d", k, v.Weight)
			tempRing.AddNode(k, v.Weight)
		}
		tempRing.Bake()
		// use the tmpMap atomic replace the global cometNodeInfoMap
		cometNodeInfoMap = tmpMap
		cometRing = tempRing
		// migrate
		if ev.Migrate {
			// TODO
			// try lock
			// if can't get lock, abandon migrate (at leaest one succeed)
			// paranaell call rpc
			// unlock
		}
		log.Debug("cometNodeInfoMap len: %d", len(cometNodeInfoMap))
	}
}

// watchNode watch a named node for leader selection when failover
func watchCometNode(conn *zk.Conn, node, fpath string, pweight int, retry, ping time.Duration, vnode int, ch chan *CometNodeEvent) {
	fpath = path.Join(fpath, node)
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err == myzk.ErrNodeNotExist {
			log.Warn("zk don't have node \"%s\"", fpath)
			break
		} else if err == myzk.ErrNoChild {
			log.Warn("zk don't have any children in \"%s\", retry in %d second", fpath, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err != nil {
			log.Error("zk path: \"%s\" getNodes error(%v), retry in %d second", fpath, err, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		}
		// leader selection
		sort.Strings(nodes)
		if info, err := registerCometNode(conn, nodes[0], fpath, retry, ping, vnode, true); err != nil {
			log.Error("zk path: \"%s\" registerCometNode error(%v)", fpath, err)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else {
			// update node info
			ch <- &CometNodeEvent{Event: eventNodeUpdate, Key: node, Value: info, Migrate: pweight != info.Weight}
		}
		// blocking receive event
		event := <-watch
		log.Info("zk path: \"%s\" receive a event: (%v)", fpath, event)
	}
	// WARN, if no persistence node and comet rpc not config
	log.Warn("zk path: \"%s\" never watch again till recreate", fpath)
}

// registerCometNode get infomation of comet node
func registerCometNode(conn *zk.Conn, node, fpath string, retry, ping time.Duration, vnode int, startPing bool) (*CometNodeInfo, error) {
	// get current node info from zookeeper
	fpath = path.Join(fpath, node)
	data, _, err := conn.Get(fpath)
	if err != nil {
		log.Error("zk.Get(\"%s\") error(%v)", fpath, err)
		return nil, err
	}
	info := &CometNodeInfo{}
	if err = json.Unmarshal(data, info); err != nil {
		log.Error("json.Unmarshal(\"%s\", nodeData) error(%v)", string(data), err)
		return nil, err
	}
	if info.RpcAddr == nil || len(info.RpcAddr) == 0 {
		log.Error("zk nodes: \"%s\" don't have rpc addr", fpath)
		return nil, ErrCometRPC
	}
	// get old node info for finding the old rpc connection
	oldInfo := cometNodeInfoMap[node]
	// init comet rpc
	clients := make(map[string]*RPCClient, len(info.RpcAddr))
	for _, addr := range info.RpcAddr {
		if oldInfo != nil && oldInfo.Rpc != nil {
			if r, ok := oldInfo.Rpc.Clients[addr]; ok && r.Client != nil {
				// reuse the rpc connection must let old client = nil, avoid reclose rpc.
				oldInfo.Rpc.Clients[addr].Client = nil
				clients[addr] = &RPCClient{Weight: 1, Addr: addr, Client: r.Client}
				continue
			}
		}
		if r, err := rpc.Dial("tcp", addr); err != nil {
			log.Error("rpc.Dial(\"%s\") error(%v)", addr, err)
			return nil, err
		} else {
			clients[addr] = &RPCClient{Weight: 1, Addr: addr, Client: r}
		}
	}
	// comet rpc use rand load balance
	lb, err := NewRandLB(clients, cometService, retry, ping, vnode, startPing)
	if err != nil {
		log.Error("NewRandLR() error(%v)", err)
		return nil, err
	}
	info.Rpc = lb
	log.Info("zk path: \"%s\" register nodes: \"%s\"", fpath, node)
	return info, nil
}

// GetComet get the node infomation under the node.
func GetComet(key string) *CometNodeInfo {
	if cometRing == nil || len(cometNodeInfoMap) == 0 {
		return nil
	}
	node := cometRing.Hash(key)
	log.Debug("cometHash hits \"%s\"", node)
	return cometNodeInfoMap[node]
}

// InitComet init a rand lb rpc for comet module.
func InitComet(conn *zk.Conn, fpath string, retry, ping time.Duration, vnode int) {
	// watch comet path
	ch := make(chan *CometNodeEvent, 1024)
	go handleCometNodeEvent(conn, fpath, retry, ping, vnode, ch)
	go watchCometRoot(conn, fpath, ch)
}
