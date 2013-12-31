package main

import (
	"errors"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/hash"
	"launchpad.net/gozk/zookeeper"
	"sort"
	"time"
)

var (
	zk        *zookeeper.Conn
	NodesHash *hash.Ketama
	NodeInfo  = make(map[string]string)
)

var ErrNoNode = errors.New("zookeeper children is nil")

func initZK() error {
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

func BeginWatchNode() error {
	nodes, err := getNodes(Conf.Zookeeper.RootPath)
	if err != nil {
		return err
	}

	NodesHash = hash.NewKetama2(nodes, 255)
	watchNodes(nodes)

	return nil
}

func GetFirstServer(node string) string {
	return NodeInfo[node]
}

func getNodes(path string) ([]string, error) {
	children, _, err := zk.Children(path)
	if err != nil {
		return nil, err
	}

	if children == nil {
		return nil, ErrNoNode
	}

	return children, nil
}

func watchNodes(nodes []string) {
	for i := 0; i < len(nodes); i++ {
		go watchFirstServer(nodes[i])
	}
}

func watchFirstServer(node string) {
	path := fmt.Sprintf("%s/%s", Conf.Zookeeper.RootPath, node)
	for {
		subNodes, err := getNodes(path)
		if err != nil {
			Log.Error("watch node:%s error (%v)", node, err)
			time.Sleep(10 * time.Second)
			continue
		}

		if len(subNodes) == 0 {
			Log.Warn("node:%s has no server", node)
			time.Sleep(10 * time.Second)
			continue
		}

		sort.Strings(subNodes)
		data, _, watch, err := zk.GetW(fmt.Sprintf("%s/%s", path, subNodes[0]))
		if err != nil {
			Log.Error("watch node:%s error (%v)", node, err)
			time.Sleep(10 * time.Second)
			continue
		}

		NodeInfo[node] = data
		Log.Info("begin to watch node:%s", node)
		event := <-watch
		NodeInfo[node] = ""
		Log.Info("end to watch node:%s event:%v", node, event)
	}
}
