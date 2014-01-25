package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"launchpad.net/gozk/zookeeper"
	"net/rpc"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	// Zookeeper connection
	zk *zookeeper.Conn
	// Ketama algorithm for Comet
	CometHash *hash.Ketama
	// Store the first alive server of every node
	// If there is no alive server under the node, the value will be nil, but key is exist in map
	NodeInfoMap = make(map[string]*NodeInfo)
	// Lock for NodeInfoMap
	NodeInfoMapLock sync.RWMutex
)

var ErrNoNode = errors.New("zookeeper children is nil")
var ErrNodeExist = errors.New("node already exist")

type NodeInfo struct {
	// The addr for subscribe
	SubAddr string
	// The connection for publish RPC
	PubRPC *rpc.Client
}

// InitWatch initialize watch module
func InitWatch() error {
	// Initialize zookeeper connection
	zkTmp, session, err := zookeeper.Dial(Conf.ZKAddr, Conf.ZKTimeout)
	if err != nil {
		return err
	}
	zk = zkTmp
	//defer zk.Close()

	for {
		event := <-session
		if event.State < zookeeper.STATE_CONNECTING {
			return errors.New(fmt.Sprintf("connect zookeeper fail, event: %v", event))
		} else if event.State == zookeeper.STATE_CONNECTING {
			time.Sleep(time.Second)
			continue
		} else {
			break
		}
	}

	return nil
}

// BeginWatchNode start watch all of nodes
func BeginWatchNode() error {
	nodes, err := getNodes(Conf.ZKRootPath)
	if err != nil {
		return err
	}

	CometHash = hash.NewKetama2(nodes, 255)
	watchNodes(nodes)

	return nil
}

// GetNode get the node infomation under the node
func GetNode(node string) *NodeInfo {
	NodeInfoMapLock.RLock()
	defer NodeInfoMapLock.RUnlock()

	return NodeInfoMap[node]
}

// AddNode add a node and watch it, and notice Comet to migrate node
func AddNode(node string) error {
	NodeInfoMapLock.RLock()
	defer NodeInfoMapLock.RUnlock()
	_, ok := NodeInfoMap[node]
	if ok {
		return ErrNodeExist
	}

	var nodes []string
	for n, _ := range NodeInfoMap {
		nodes = append(nodes, n)
	}

	nodes = append(nodes, node)

	// Notice Comet to migrate node
	if err := ChannelRPCMigrate(nodes, NodeInfoMap); err != nil {
		return err
	}

	// Watch the node
	go watchFirstServer(node)

	// Update Comet hash, because of nodes are changed
	CometHash = hash.NewKetama2(nodes, 255)

	return nil
}

// DelNode disconnect and delete a node, and notice Comet to migrate node
func DelNode(node string) error {
	var (
		nodes []string
		info  *NodeInfo
	)

	NodeInfoMapLock.Lock()
	defer NodeInfoMapLock.Unlock()

	if _, ok := NodeInfoMap[node]; !ok {
		return nil
	}

	for n, c := range NodeInfoMap {
		if n == node {
			info = c
			continue
		}

		nodes = append(nodes, n)
	}

	// Update Comet hash, cause nodes are changed
	CometHash = hash.NewKetama2(nodes, 255)

	// Delete node from map before call Migrate RPC interface of Comet, cause needn`t to notice deleted node
	delete(NodeInfoMap, node)

	if info != nil && info.PubRPC != nil {
		info.PubRPC.Close()
		info.PubRPC = nil
	}

	// Notice Comet to migrate node
	if err := ChannelRPCMigrate(nodes, NodeInfoMap); err != nil {
		return err
	}

	return nil
}

// RPC Migrate interface
// Migrate the lost connections after changed node
func ChannelRPCMigrate(nodes []string, nodeInfoMap map[string]*NodeInfo) error {
	ret := OK

	for n, svrInfo := range nodeInfoMap {
		if svrInfo != nil && svrInfo.PubRPC != nil {
			args := &myrpc.ChannelMigrateArgs{Nodes: nodes, Vnode: 255}
			if err := svrInfo.PubRPC.Call("ChannelRPC.Migrate", args, &ret); err != nil {
				Log.Error("RPC.Call(\"ChannelRPC.Migrate\") error node:%s (%v)", n, err)
				return err
			}

			if ret != OK {
				err := errors.New(fmt.Sprintf("ret:%d", ret))
				Log.Error("RPC.Call(\"ChannelRPC.Migrate\") error node:%s (%v)", n, err)
				return err
			}

			Log.Debug("RPC.Call(\"ChannelRPC.Migrate\") success node:%s", n)
		}
	}

	return nil
}

// getNodes get all of nodes under the path
func getNodes(path string) ([]string, error) {
	children, _, err := zk.Children(path)
	if err != nil {
		Log.Error("zk.Children(%s) error", path)
		return nil, err
	}

	if children == nil {
		return nil, ErrNoNode
	}

	return children, nil
}

// getNodesW get all of nodes with watch under the path
func getNodesW(path string) ([]string, <-chan zookeeper.Event, error) {
	children, _, watch, err := zk.ChildrenW(path)
	if err != nil {
		Log.Error("zk.Children(%s) error", path)
		return nil, nil, err
	}

	if children == nil {
		return nil, nil, ErrNoNode
	}

	return children, watch, nil
}

// watchNodes watch all of nodes
func watchNodes(nodes []string) {
	for i := 0; i < len(nodes); i++ {
		go watchFirstServer(nodes[i])
	}
}

// watchNodes watch the first server under the node, and keep rpc with publish RPC
// the first server must be alive
func watchFirstServer(node string) {
	path := fmt.Sprintf("%s/%s", Conf.ZKRootPath, node)
	for {
		subNodes, watch, err := getNodesW(path)
		if err != nil {
			Log.Error("watch node:%s error (%v)", node, err)
			time.Sleep(10 * time.Second)
			continue
		}

		// If exist server then set it into NodeInfoMap
		if len(subNodes) != 0 {
			sort.Strings(subNodes)
			data, _, err := zk.Get(fmt.Sprintf("%s/%s", path, subNodes[0]))
			if err != nil {
				Log.Error("watch node:%s, subNode:%s, error (%v)", node, subNodes[0], err)
				time.Sleep(10 * time.Second)
				continue
			}

			// Fecth push server info
			datas := strings.Split(data, ",")
			if len(datas) < 2 {
				Log.Error("get subNode data error node:%s, subNode:%s, data:%s", node, subNodes[0], data)
				time.Sleep(10 * time.Second)
				continue
			}

			NodeInfoMapLock.RLock()
			info, ok := NodeInfoMap[node]
			if ok {
				info.SubAddr = datas[0]
				if info.PubRPC != nil {
					info.PubRPC.Close()
				}
			} else {
				info = &NodeInfo{SubAddr: datas[0]}
			}
			NodeInfoMapLock.RUnlock()

			// ReDial RPC
			r, err := rpc.Dial(Conf.CometNetwork, datas[1])
			if err != nil {
				Log.Error("rpc.Dial(%s, %s) error node:%s, subNode:%s", Conf.CometNetwork, datas[1], node, subNodes[0])
				time.Sleep(10 * time.Second)
				continue
			}

			info.PubRPC = r
			NodeInfoMapLock.Lock()
			NodeInfoMap[node] = info
			NodeInfoMapLock.Unlock()
		} else {
			NodeInfoMapLock.Lock()
			NodeInfoMap[node] = nil
			NodeInfoMapLock.Unlock()
		}

		Log.Warn("begin to watch node:%s", node)
		event := <-watch
		if event.Type == zookeeper.EVENT_DELETED {
			Log.Warn("stop to watch node:%s", node)
			DelNode(node)
			break
		}

		Log.Warn("end to watch node:%s event:%v, try to watch repeated", node, event)
	}
}
