package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	timeID "github.com/Terry-Mao/gopush-cluster/id"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"launchpad.net/gozk/zookeeper"
	"net/rpc"
	"sort"
	"strings"
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
)

var ErrNoChild = errors.New("zookeeper children is nil")
var ErrNodeExist = errors.New("node exist")

// Protocol of Comet subcription
const (
	ProtocolUnknown = 0
	ProtocolWS      = 1
	ProtocolWSStr   = "ws"
	ProtocolTCP     = 2
	ProtocolTCPStr  = "tcp"
	ProtocolRPC     = 3
	ProtocolRPCStr  = "rpc"
)

type NodeInfo struct {
	// The addr for subscribe, format like:map[Protocol]Addr
	SubAddr map[int]string
	// The connection for Comet RPC
	PubRPC *rpc.Client
}

// InitWatch initialize watch module
func InitWatch() error {
	// Initialize zookeeper connection
	Log.Info("Initializing zookeeper,zookeeper.Dial(\"%s\", \"%dms\")", Conf.ZKAddr, Conf.ZKTimeout/1000000)
	zkTmp, session, err := zookeeper.Dial(Conf.ZKAddr, Conf.ZKTimeout)
	if err != nil {
		return err
	}
	zk = zkTmp

	// Zookeeper client will reconnecting automatically
	for {
		event := <-session
		if event.State < zookeeper.STATE_CONNECTING {
			return errors.New(fmt.Sprintf("connect zookeeper fail, event:\"%v\"", event))
		} else if event.State == zookeeper.STATE_CONNECTING {
			Log.Warn("Zookeeper connecting!")
			time.Sleep(time.Second)
			continue
		} else {
			break
		}
	}

	// Zookeeper create Public message subnode
	if err := zkCreate(); err != nil {
		return err
	}
	Log.Info("Initialize zookeeper OK")

	// Init public message mid-creater
	PubMID = timeID.NewTimeID()

	return nil
}

// WatchStop stop watch
func WatchStop() {
	if zk != nil {
		if err := zk.Close(); err != nil {
			Log.Error("zk.Close() error(%v)", err)
		}
	}
}

// zkCreate zookeeper init subnode
func zkCreate() error {
	// Create zk public message lock root path
	Log.Debug("create zookeeper path:%s", Conf.ZKPIDPath)
	_, err := zk.Create(Conf.ZKPIDPath, "", 0, zookeeper.WorldACL(zookeeper.PERM_ALL))
	if err != nil {
		if zookeeper.IsError(err, zookeeper.ZNODEEXISTS) {
			Log.Warn("zk.create(\"%s\") exists", Conf.ZKPIDPath)
		} else {
			Log.Error("zk.create(\"%s\") error(%v)", Conf.ZKPIDPath, err)
			return err
		}
	}
	return nil
}

// BeginWatchNode start watch all of nodes which registered in zookeeper
func BeginWatchNode() error {
	nodes, err := getNodes(Conf.ZKCometPath)
	if err != nil {
		Log.Error("getNodes(\"%s\") error(%v)", Conf.ZKCometPath, err)
		return err
	}

	for _, n := range nodes {
		NodeInfoMap[n] = nil
	}

	// Update Comet hash
	CometHash = hash.NewKetama2(nodes, 255)
	// Watch all of nodes
	watchNodes(nodes)

	return nil
}

// GetNode get the node infomation under the node
func GetNode(node string) *NodeInfo {
	return NodeInfoMap[node]
}

// GetNode get the node quantity
func NodeQuantity() int {
	return len(NodeInfoMap)
}

// AddNode add a node and watch it, and notice Comet to migrate node
func AddNode(node string) error {
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
	if err := channelRPCMigrate(nodes, NodeInfoMap); err != nil {
		return err
	}

	// Watch the node
	go watchFirstService(node)

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

	if _, ok := NodeInfoMap[node]; !ok {
		return nil
	}

	// Delete node from map before call Migrate RPC interface of Comet, cause needn`t to notice deleted node
	tmpMap := make(map[string]*NodeInfo)
	for n, i := range NodeInfoMap {
		if n == node {
			info = i
			continue
		}

		tmpMap[n] = i
		nodes = append(nodes, n)
	}
	NodeInfoMap = tmpMap

	// Update Comet hash, cause nodes are changed
	CometHash = hash.NewKetama2(nodes, 255)

	if info != nil && info.PubRPC != nil {
		info.PubRPC.Close()
	}

	// Notice Comet to migrate node
	if err := channelRPCMigrate(nodes, NodeInfoMap); err != nil {
		return err
	}

	return nil
}

// RPC Migrate interface
// Migrate the lost connections after changed node
func channelRPCMigrate(nodes []string, nodeInfoMap map[string]*NodeInfo) error {
	ret := OK

	for n, svrInfo := range nodeInfoMap {
		if svrInfo != nil && svrInfo.PubRPC != nil {
			args := &myrpc.ChannelMigrateArgs{Nodes: nodes, Vnode: 255}
			if err := svrInfo.PubRPC.Call("ChannelRPC.Migrate", args, &ret); err != nil {
				Log.Error("RPC.Call(\"ChannelRPC.Migrate\") error node:\"%s\" error(%v)", n, err)
				return err
			}

			if ret != OK {
				err := errors.New(fmt.Sprintf("ret:%d", ret))
				Log.Error("RPC.Call(\"ChannelRPC.Migrate\") error node:\"%s\" errorCode:\"%d\"", n, ret)
				return err
			}

			Log.Info("RPC.Call(\"ChannelRPC.Migrate\") success node:\"%s\"", n)
		}
	}

	return nil
}

// getNodes get all of nodes under the path
func getNodes(path string) ([]string, error) {
	children, _, err := zk.Children(path)
	if err != nil {
		return nil, err
	}

	if children == nil {
		return nil, ErrNoChild
	}

	return children, nil
}

// getNodesW get all of nodes with watch under the path
func getNodesW(path string) ([]string, <-chan zookeeper.Event, error) {
	children, _, watch, err := zk.ChildrenW(path)
	if err != nil {
		return nil, nil, err
	}

	if children == nil {
		return nil, nil, ErrNoChild
	}

	return children, watch, nil
}

// watchNodes watch all of nodes
func watchNodes(nodes []string) {
	for i := 0; i < len(nodes); i++ {
		go watchFirstService(nodes[i])
	}
}

// watchNodes watch the first service under the node, and keep rpc connecting with Comet RPC,
// the first service must be alive
func watchFirstService(node string) {
	defer func() {
		if err := recover(); err != nil {
			Log.Error("watching node goroutine panic node:\"%s\" error(%v), stop watching", node, err)

			var nodes []string
			// Delete node from map
			tmpMap := make(map[string]*NodeInfo)
			for n, i := range NodeInfoMap {
				if n == node {
					continue
				}
				tmpMap[n] = i
				nodes = append(nodes, n)
			}
			NodeInfoMap = tmpMap

			// Update Comet hash, cause nodes are changed
			CometHash = hash.NewKetama2(nodes, 255)
		}
	}()
	path := fmt.Sprintf("%s/%s", Conf.ZKCometPath, node)
	for {
		subNodes, watch, err := getNodesW(path)
		if err != nil {
			// If no subNode, then recheck repeatedly
			if err == ErrNoChild {
				Log.Warn("get service of node:\"%s\" error(%v), recheck after 5 seconds", node, err)
				time.Sleep(5 * time.Second)
				continue
			}

			Log.Error("get service of node:\"%s\" error(%v), stop watching", node, err)
			break
		}

		// Get service infomation
		sort.Strings(subNodes)
		data, _, err := zk.Get(fmt.Sprintf("%s/%s", path, subNodes[0]))
		if err != nil {
			Log.Error("watch node:\"%s\", subNode:\"%s\", error(%v)", node, subNodes[0], err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Fecth and parse push service connection info
		subAddr, err := parseZKData(data)
		if err != nil {
			Log.Error("get subNode data error node:\"%s\", subNode:\"%s\", error(%v)", node, subNodes[0], err)
			time.Sleep(5 * time.Second)
			continue
		}
		info := NodeInfoMap[node]
		if info != nil {
			info.SubAddr = subAddr
			if info.PubRPC != nil {
				info.PubRPC.Close()
			}
		} else {
			info = &NodeInfo{SubAddr: subAddr}
		}
		tmpMap := make(map[string]*NodeInfo)
		for n, i := range NodeInfoMap {
			tmpMap[n] = i
		}
		tmpMap[node] = info
		NodeInfoMap = tmpMap
		// ReDial RPC
		addr, ok := subAddr[ProtocolRPC]
		if ok {
			r, err := rpc.Dial("tcp", addr)
			if err != nil {
				Log.Error("rpc.Dial(\"tcp\", \"%s\") error(%v) node:\"%s\", subNode:\"%s\", recheck after 5 seconds", addr, err, node, subNodes[0])
				time.Sleep(5 * time.Second)
				continue
			}

			info.PubRPC = r
		} else {
			Log.Error("rpc.Dial(\"tcp\", \"%s\") error node:\"%s\", subNode:\"%s\" error(no rpc address)", addr, node, subNodes[0])
		}

		Log.Info("begin watching node:\"%s\" addr:\"%v\"", node, subAddr)
		event := <-watch
		Log.Warn("end watching node:\"%s\" addr:\"%v\" event:\"%v\", retry to watch", node, subAddr, event)
	}

	// Delete node
	Log.Warn("stop watching node:\"%s\"", node)
	if err := DelNode(node); err != nil {
		Log.Error("stop watching node:\"%s\" error(%v)", node, err)
		return
	}
}

// parseZKData parse the protocol data, the data format like: ws://ip:port1,tcp://ip:port2,rpc://ip:port3
func parseZKData(zkData string) (map[int]string, error) {
	res := make(map[int]string)
	data := strings.Split(zkData, ",")
	for i := 0; i < len(data); i++ {
		addr := strings.Split(data[i], "://")
		if len(addr) != 2 {
			return nil, fmt.Errorf("data:\"%s\" format error", data)
		}

		res[getProtocolInt(addr[0])] = addr[1]
	}

	return res, nil
}

// getProtocolInt get the figure corresponding with protocol string
func getProtocolInt(protocol string) int {
	if protocol == ProtocolWSStr {
		return ProtocolWS
	} else if protocol == ProtocolTCPStr {
		return ProtocolTCP
	} else if protocol == ProtocolRPCStr {
		return ProtocolRPC
	} else {
		return ProtocolUnknown
	}
}
