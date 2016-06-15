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
	log "github.com/alecthomas/log4go"
	"encoding/json"
	"errors"
	"github.com/Terry-Mao/gopush-cluster/ketama"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"path"
	"sort"
	"sync"
	"time"
)

const (
	cometService             = "CometRPC"
	CometServicePushPrivate  = "CometRPC.PushPrivate"
	CometServicePushPrivates = "CometRPC.PushPrivates"
	CometServiceMigrate      = "CometRPC.Migrate"
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
	RpcAddr []string `json:"rpc"`
	TcpAddr []string `json:"tcp"`
	WsAddr  []string `json:"ws"`
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
				ch <- &CometNodeEvent{Event: eventNodeDel, Key: node}
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
				ch <- &CometNodeEvent{Event: eventNodeAdd, Key: node}
			}
			nodesMap[node] = true
		}
		// handle delete nodes
		for node, _ := range cometNodeInfoMap {
			if _, ok := nodesMap[node]; !ok {
				ch <- &CometNodeEvent{Event: eventNodeDel, Key: node}
			}
		}
		event := <-watch
		log.Info("zk path: \"%s\" receive a event %v", fpath, event)
	}
}

// handleCometNodeEvent add and remove CometNodeInfo, copy the src map to a new map then replace the variable.
func handleCometNodeEvent(conn *zk.Conn, migrateLockPath, fpath string, retry, ping time.Duration, ch chan *CometNodeEvent) {
	for {
		ev := <-ch
		var (
			update = false
			znode  = path.Join(fpath, ev.Key)
		)
		// copy map from src
		tmpMap := make(map[string]*CometNodeInfo, len(cometNodeInfoMap))
		for k, v := range cometNodeInfoMap {
			tmpMap[k] = v
		}
		// handle event
		if ev.Event == eventNodeAdd {
			log.Info("add node: \"%s\"", ev.Key)
			tmpMap[ev.Key] = &CometNodeInfo{Weight: 1}
			go watchCometNode(conn, ev.Key, fpath, retry, ping, ch)
		} else if ev.Event == eventNodeDel {
			log.Info("del node: \"%s\"", ev.Key)
			delete(tmpMap, ev.Key)
		} else if ev.Event == eventNodeUpdate {
			log.Info("update node: \"%s\"", ev.Key)
			// when new node added to watchCometNode then trigger node update
			tmpMap[ev.Key] = ev.Value
			update = true
		} else {
			log.Error("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		// if exist old node info, destroy
		// if node add this may not happan
		// if node del this will clean the resource
		// if node update, after reuse rpc connection, this will clean the resource
		if info, ok := cometNodeInfoMap[ev.Key]; ok {
			if info != nil && info.Rpc != nil {
				info.Rpc.Destroy()
			}
		}
		// update comet hash, cause node has changed
		tempRing := ketama.NewRing(ketama.Base)
		nodeWeightMap := map[string]int{}
		for k, v := range tmpMap {
			log.Debug("AddNode node:%s weight:%d", k, v.Weight)
			tempRing.AddNode(k, v.Weight)
			nodeWeightMap[k] = v.Weight
		}
		tempRing.Bake()
		// use the tmpMap atomic replace the global cometNodeInfoMap
		cometNodeInfoMap = tmpMap
		cometRing = tempRing
		// migrate
		if ev.Event != eventNodeAdd {
			if err := notifyMigrate(conn, migrateLockPath, znode, ev.Key, update, nodeWeightMap); err != nil {
				// if err == zk.ErrNodeExists meaning anyone is going through.
				// we hopefully that only one web node notify comet migrate.
				// also it was judged in Comet whether it needs migrate or not.
				if err == zk.ErrNodeExists {
					log.Info("ignore notify migrate")
					continue
				} else {
					log.Error("notifyMigrate(conn, \"%v\") error(%v)", nodeWeightMap, err)
					continue
				}
			}
		}
		log.Debug("cometNodeInfoMap len: %d", len(cometNodeInfoMap))
	}
}

// notify every Comet node to migrate
func notifyMigrate(conn *zk.Conn, migrateLockPath, znode, key string, update bool, nodeWeightMap map[string]int) (err error) {
	// try lock
	if _, err = conn.Create(migrateLockPath, []byte("1"), zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil {
		log.Error("conn.Create(\"/gopush-migrate-lock\", \"1\", zk.FlagEphemeral) error(%v)", err)
		return
	}
	// call comet migrate rpc
	wg := &sync.WaitGroup{}
	wg.Add(len(cometNodeInfoMap))
	for node, nodeInfo := range cometNodeInfoMap {
		go func(n string, info *CometNodeInfo) {
			if info.Rpc == nil {
				log.Error("notify migrate failed, no rpc found, node:%s", n)
				wg.Done()
				return
			}
			r := info.Rpc.Get()
			if r == nil {
				log.Error("notify migrate failed, no rpc found, node:%s", n)
				wg.Done()
				return
			}
			reply := 0
			args := &CometMigrateArgs{Nodes: nodeWeightMap}
			if err = r.Call(CometServiceMigrate, args, &reply); err != nil {
				log.Error("rpc.Call(\"%s\") error(%v)", CometServiceMigrate, err)
				wg.Done()
				return
			}
			log.Debug("notify node:%s migrate succeed", n)
			wg.Done()
		}(node, nodeInfo)
	}
	wg.Wait()
	// update znode info
	if update {
		var data []byte
		data, err = json.Marshal(cometNodeInfoMap[key])
		if err != nil {
			log.Error("json.Marshal() node:%s error(%v)", key, err)
			return
		}
		if _, err = conn.Set(znode, data, -1); err != nil {
			log.Error("conn.Set(\"%s\",\"%s\",\"-1\") error(%v)", znode, string(data), err)
			return
		}
	}

	// release lock
	if err = conn.Delete(migrateLockPath, -1); err != nil {
		log.Error("conn.Delete(\"%s\") error(%v)", migrateLockPath, err)
	}
	return
}

// watchNode watch a named node for leader selection when failover
func watchCometNode(conn *zk.Conn, node, fpath string, retry, ping time.Duration, ch chan *CometNodeEvent) {
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
		if info, err := registerCometNode(conn, nodes[0], fpath, retry, ping, true); err != nil {
			log.Error("zk path: \"%s\" registerCometNode error(%v)", fpath, err)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else {
			// update node info
			ch <- &CometNodeEvent{Event: eventNodeUpdate, Key: node, Value: info}
		}
		// blocking receive event
		event := <-watch
		log.Info("zk path: \"%s\" receive a event: (%v)", fpath, event)
	}
	// WARN, if no persistence node and comet rpc not config
	log.Warn("zk path: \"%s\" never watch again till recreate", fpath)
}

// registerCometNode get infomation of comet node
func registerCometNode(conn *zk.Conn, node, fpath string, retry, ping time.Duration, startPing bool) (info *CometNodeInfo, err error) {
	// get current node info from zookeeper
	fpath = path.Join(fpath, node)
	data, _, err := conn.Get(fpath)
	if err != nil {
		log.Error("zk.Get(\"%s\") error(%v)", fpath, err)
		return
	}
	info = &CometNodeInfo{}
	if err = json.Unmarshal(data, info); err != nil {
		log.Error("json.Unmarshal(\"%s\", nodeData) error(%v)", string(data), err)
		return
	}
	if len(info.RpcAddr) == 0 {
		log.Error("zk nodes: \"%s\" don't have rpc addr", fpath)
		err = ErrCometRPC
		return
	}
	// get old node info for finding the old rpc connection
	oldInfo := cometNodeInfoMap[node]
	// init comet rpc
	clients := make(map[string]*WeightRpc, len(info.RpcAddr))
	for _, addr := range info.RpcAddr {
		var (
			r *rpc.Client
		)
		if oldInfo != nil && oldInfo.Rpc != nil {
			if wr, ok := oldInfo.Rpc.Clients[addr]; ok && wr.Client != nil {
				// reuse the rpc connection must let old client = nil, avoid reclose rpc.
				oldInfo.Rpc.Clients[addr].Client = nil
				r = wr.Client
			}
		}
		if r == nil {
			if r, err = rpc.Dial("tcp", addr); err != nil {
				log.Error("rpc.Dial(\"%s\") error(%v)", addr, err)
				return
			}
			log.Debug("node:%s addr:%s rpc reconnect", node, addr)
		}
		clients[addr] = &WeightRpc{Weight: 1, Addr: addr, Client: r}
	}
	// comet rpc use rand load balance
	lb, err := NewRandLB(clients, cometService, retry, ping, startPing)
	if err != nil {
		log.Error("NewRandLR() error(%v)", err)
		return
	}
	info.Rpc = lb
	log.Info("zk path: \"%s\" register nodes: \"%s\"", fpath, node)
	return
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
func InitComet(conn *zk.Conn, migrateLockPath, fpath string, retry, ping time.Duration) {
	// watch comet path
	ch := make(chan *CometNodeEvent, 1024)
	go handleCometNodeEvent(conn, migrateLockPath, fpath, retry, ping, ch)
	go watchCometRoot(conn, fpath, ch)
}
