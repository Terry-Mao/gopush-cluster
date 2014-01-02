package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"launchpad.net/gozk/zookeeper"
	"net/rpc"
	"sort"
	"strings"
	"time"
)

var (
	// zookeeper connection
	zk *zookeeper.Conn
	// ketama algorithm
	CometHash *hash.Ketama
	// Store the first alive server of every node
	// If there is no alive server under node, the value will be "", but key is exist in map
	NodeInfoMap = make(map[string]*NodeInfo)
)

var ErrNoNode = errors.New("zookeeper children is nil")

type NodeInfo struct {
	// The addr for subscribe
	Data string
	// The connection for publish RPC
	PubRPC *rpc.Client
}

// InitWatch initialize watch module
func InitWatch() error {
	// Initialize zookeeper connection
	zkTmp, session, err := zookeeper.Dial(Conf.Zookeeper.Addr, time.Duration(Conf.Zookeeper.Timeout)*1e9)
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
	nodes, err := getNodes(Conf.Zookeeper.RootPath)
	if err != nil {
		return err
	}

	CometHash = hash.NewKetama2(nodes, 255)
	watchNodes(nodes)

	return nil
}

// GetFirstServer get the first server under the node
func GetFirstServer(node string) *NodeInfo {
	return NodeInfoMap[node]
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
	path := fmt.Sprintf("%s/%s", Conf.Zookeeper.RootPath, node)
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
				Log.Error("subNode data error node:%s, subNode:%s", node, subNodes[0])
				time.Sleep(10 * time.Second)
				continue
			}

			info, ok := NodeInfoMap[node]
			if ok {
				info.Data = datas[0]
				if info.PubRPC != nil {
					info.PubRPC.Close()
				}
			} else {
				info = &NodeInfo{Data: datas[0]}
			}

			// ReDial RPC
			r, err := rpc.Dial(Conf.Push.Network, datas[1])
			if err != nil {
				Log.Error("rpc.Dial(%s, %s) error node:%s, subNode:%s", Conf.Push.Network, datas[1], node, subNodes[0])
				time.Sleep(10 * time.Second)
				continue
			}

			info.PubRPC = r
			NodeInfoMap[node] = info
		} else {
			NodeInfoMap[node] = nil
		}

		Log.Info("begin to watch node:%s", node)
		event := <-watch
		Log.Info("end to watch node:%s event:%v", node, event)
	}
}
