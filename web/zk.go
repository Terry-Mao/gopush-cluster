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
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"net/rpc"
	"sort"
	"strings"
	"time"
)

const (
	// protocol of Comet subcription
	protocolUnknown = 0
	protocolWS      = 1
	protocolWSStr   = "ws"
	protocolTCP     = 2
	protocolTCPStr  = "tcp"
	protocolRPC     = 3
	protocolRPCStr  = "rpc"

	// node event
	eventNodeAdd    = 1
	eventNodeDel    = 2
	eventNodeUpdate = 3

	// wait node
	waitNodeDelay       = 5
	waitNodeDelaySecond = waitNodeDelay * time.Second
)

var (
	// error
	errNoChild      = errors.New("zk: children is nil")
	errNodeNotExist = errors.New("zk: node not exist")
	errCometRPC     = errors.New("comet rpc call failed")

	// Store the first alive Comet service of every node
	// If there is no alive service under the node, the map`s value will be nil, but key is exist in map
	NodeInfoMap = make(map[string]*NodeInfo)
	// Ketama algorithm for check Comet node
	cometHash *hash.Ketama
)

type NodeInfo struct {
	// The addr for subscribe, format like:map[Protocol]Addr
	Addr map[int][]string
	// The connection for Comet RPC
	PubRPC *rpc.Client
}

// Close close the node rpc connection.
func (n *NodeInfo) Close() {
	if n.PubRPC != nil {
		if err := n.PubRPC.Close(); err != nil {
			glog.Errorf("rpc.Close() error(%v)", err)
		}
	}
}

type NodeEvent struct {
	// node name(node1, node2...)
	Key string
	// node info
	Value *NodeInfo
	// event type
	Event int
}

func InitZK() (*zk.Conn, error) {
	// connect to zookeeper, get event from chan in goroutine(log)
	glog.V(1).Infof("zk timeout: %d", Conf.ZKTimeout)
	conn, session, err := zk.Connect(Conf.ZKAddr, Conf.ZKTimeout)
	if err != nil {
		glog.Errorf("zk.Connect(\"%v\", %d) error(%v)", Conf.ZKAddr, Conf.ZKTimeout, err)
		return nil, err
	}
	go func() {
		for {
			event := <-session
			glog.Infof("zookeeper get a event: %s", event.State.String())
		}
	}()
	// Create zk public message-lock root path
	glog.V(1).Infof("create zookeeper path: %s", Conf.ZKPIDPath)
	_, err = conn.Create(Conf.ZKPIDPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists {
			glog.Warningf("zk.create(\"%s\") exists", Conf.ZKPIDPath)
		} else {
			glog.Errorf("zk.create(\"%s\") error(%v)", Conf.ZKPIDPath, err)
			return nil, err
		}
	}
	// after init nodes, then watch them
	ch := make(chan *NodeEvent, 1024)
	go handleNodeEvent(conn, Conf.ZKCometPath, ch)
	go watchRoot(conn, Conf.ZKCometPath, ch)
	return conn, nil
}

func watchRoot(conn *zk.Conn, path string, ch chan *NodeEvent) error {
	for {
		nodes, watch, err := getNodesW(conn, path)
		if err == errNodeNotExist {
			glog.Warningf("zk don't have node \"%s\", retry in %d second", path, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err == errNoChild {
			glog.Warningf("zk don't have any children in \"%s\", retry in %d second", path, waitNodeDelay)
			for node, _ := range NodeInfoMap {
				ch <- &NodeEvent{Event: eventNodeDel, Key: node}
			}
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err != nil {
			glog.Errorf("getNodes error(%v), retry in %d second", err, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		}
		nodesMap := make(map[string]bool, len(nodes))
		// handle new add nodes
		for _, node := range nodes {
			if _, ok := NodeInfoMap[node]; !ok {
				ch <- &NodeEvent{Event: eventNodeAdd, Key: node}
			}
			nodesMap[node] = true
		}
		// handle delete nodes
		for node, _ := range NodeInfoMap {
			if _, ok := nodesMap[node]; !ok {
				ch <- &NodeEvent{Event: eventNodeDel, Key: node}
			}
		}
		// blocking wait node changed
		event := <-watch
		glog.Infof("zk path: \"%s\" receive a event %v", path, event)
	}
}

// watchNode watch a named node for leader selection when failover
func watchNode(conn *zk.Conn, node, path string, ch chan *NodeEvent) {
	path = fmt.Sprintf("%s/%s", path, node)
	for {
		nodes, watch, err := getNodesW(conn, path)
		if err == errNodeNotExist {
			glog.Warningf("zk don't have node \"%s\"", path)
			break
		} else if err == errNoChild {
			glog.Warningf("zk don't have any children in \"%s\", retry in %d second", path, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else if err != nil {
			glog.Errorf("zk path: \"%s\" getNodes error(%v), retry in %d second", path, err, waitNodeDelay)
			time.Sleep(waitNodeDelaySecond)
			continue
		}
		// leader selection
		sort.Strings(nodes)
		// register node
		if info, err := registerNode(conn, nodes[0], path); err != nil {
			glog.Errorf("zk path: \"%s\" registerNode error(%v)", path, err)
			time.Sleep(waitNodeDelaySecond)
			continue
		} else {
			// update node info
			ch <- &NodeEvent{Event: eventNodeUpdate, Key: node, Value: info}
		}
		// blocking receive event
		event := <-watch
		glog.Infof("zk path: \"%s\" receive a event: (%v)", path, event)
	}
	// WARN, if no persistence node and comet rpc not config
	glog.Warningf("zk path: \"%s\" never watch again till recreate", path)
}

func registerNode(conn *zk.Conn, node, path string) (*NodeInfo, error) {
	path = path + "/" + node
	data, _, err := conn.Get(path)
	if err != nil {
		glog.Errorf("zk.Get(\"%s\") error(%v)", path, err)
		return nil, err
	}
	// fetch and parse comet info
	addr, err := parseNode(string(data))
	if err != nil {
		glog.Errorf("parseNode(\"%s\") error(%v)", string(data), err)
		return nil, err
	}
	info := &NodeInfo{Addr: addr}
	rpcAddr, ok := addr[protocolRPC]
	if !ok || len(rpcAddr) == 0 {
		glog.Errorf("zk nodes: \"%s\" don't have rpc addr", path)
		return nil, errCometRPC
	}
	// init comet rpc
	// TODO support many rpc
	r, err := rpc.Dial("tcp", rpcAddr[0])
	if err != nil {
		glog.Errorf("rpc.Dial(\"%s\") error(%v)", rpcAddr[0], err)
		return nil, errCometRPC
	}
	info.PubRPC = r
	glog.Infof("zk path: \"%s\" register nodes: \"%s\"", path, node)
	return info, nil
}

// handleNodeEvent add and remove NodeInfo, copy the src map to a new map then replace the variable.
func handleNodeEvent(conn *zk.Conn, path string, ch chan *NodeEvent) {
	for {
		ev := <-ch
		// copy map from src
		tmpMap := make(map[string]*NodeInfo, len(NodeInfoMap))
		for k, v := range NodeInfoMap {
			tmpMap[k] = v
		}
		// handle event
		if ev.Event == eventNodeAdd {
			glog.Infof("add node: \"%s\"", ev.Key)
			tmpMap[ev.Key] = nil
			go watchNode(conn, ev.Key, path, ch)
		} else if ev.Event == eventNodeDel {
			glog.Infof("del node: \"%s\"", ev.Key)
			delete(tmpMap, ev.Key)
		} else if ev.Event == eventNodeUpdate {
			glog.Infof("update node: \"%s\"", ev.Key)
			tmpMap[ev.Key] = ev.Value
		} else {
			glog.Errorf("unknown node event: %d", ev.Event)
			panic("unknown node event")
		}
		// if exist old node info, close
		if info, ok := NodeInfoMap[ev.Key]; ok {
			if info != nil {
				info.Close()
				glog.Infof("close old node info: \"%s\"", ev.Key)
			}
		}
		// use the tmpMap atomic replace the global NodeInfoMap
		NodeInfoMap = tmpMap
		// update comet hash, cause node has changed
		nodes := make([]string, 0, len(tmpMap))
		for k, _ := range tmpMap {
			nodes = append(nodes, k)
		}
		cometHash = hash.NewKetama2(nodes, 255)
		glog.V(1).Infof("NodeInfoMap len: %d", len(NodeInfoMap))
	}
}

// get all child from zk path
func getNodes(conn *zk.Conn, path string) ([]string, error) {
	nodes, stat, err := conn.Children(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, errNodeNotExist
		}
		glog.Errorf("zk.Children(\"%s\") error(%v)", path)
		return nil, err
	}
	if stat == nil {
		return nil, errNodeNotExist
	}
	if len(nodes) == 0 {
		return nil, errNoChild
	}
	return nodes, nil
}

// get all child from zk path with a watch
func getNodesW(conn *zk.Conn, path string) ([]string, <-chan zk.Event, error) {
	nodes, stat, watch, err := conn.ChildrenW(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, errNodeNotExist
		}
		glog.Errorf("zk.Children(\"%s\") error(%v)", path, err)
		return nil, nil, err
	}
	if stat == nil {
		return nil, nil, errNodeNotExist
	}
	if len(nodes) == 0 {
		return nil, nil, errNoChild
	}
	return nodes, watch, nil
}

// FindNode get the node infomation under the node.
func FindNode(key string) *NodeInfo {
	if cometHash == nil || len(NodeInfoMap) == 0 {
		return nil
	}
	node := cometHash.Node(key)
	glog.V(1).Infof("cometHash hits \"%s\"", node)
	return NodeInfoMap[node]
}

// parseNode parse the protocol data, the data format like: ws://ip:port1,tcp://ip:port2,rpc://ip:port3
func parseNode(data string) (map[int][]string, error) {
	res := make(map[int][]string)
	dataArr := strings.Split(data, ",")
	for i := 0; i < len(dataArr); i++ {
		addrArr := strings.Split(dataArr[i], "://")
		if len(addrArr) != 2 {
			return nil, fmt.Errorf("data:\"%s\" format error", data)
		}
		key := protoInt(addrArr[0])
		val, ok := res[key]
		if ok {
			val = append(val, addrArr[1])
		} else {
			val = []string{addrArr[1]}
		}
		res[key] = val
	}
	return res, nil
}

// protoInt get the figure corresponding with protocol string
func protoInt(protocol string) int {
	if protocol == protocolWSStr {
		return protocolWS
	} else if protocol == protocolTCPStr {
		return protocolTCP
	} else if protocol == protocolRPCStr {
		return protocolRPC
	} else {
		return protocolUnknown
	}
}
