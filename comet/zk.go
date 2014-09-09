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
	log "code.google.com/p/log4go"
	"fmt"
	"github.com/Terry-Mao/gopush-cluster/rpc"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/samuel/go-zookeeper/zk"
	"path"
	"strings"
	"time"
)

var cometNodeInfoMap = make(map[string]*myrpc.CometNodeInfo)

const (
	// wait node
	waitNodeDelay       = 3
	waitNodeDelaySecond = waitNodeDelay * time.Second
)

func InitZK() (*zk.Conn, error) {
	conn, err := myzk.Connect(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		log.Error("myzk.Connect() error(%v)", err)
		return nil, err
	}
	fpath := path.Join(Conf.ZookeeperCometPath, Conf.ZookeeperCometNode)
	if err = myzk.Create(conn, fpath, Conf.ZookeeperCometWeight); err != nil {
		log.Error("myzk.CreateWithData(\"%s\",\"%s\") error(%v)", fpath, Conf.ZookeeperCometWeight, err)
		return conn, err
	}
	// comet websocket and rpc bind address store in the zk
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
	data = strings.TrimRight(data, ",")
	log.Debug("myzk node:\"%s\" registe data: \"%s\"", fpath, data)
	if err = myzk.RegisterTemp(conn, fpath, data); err != nil {
		log.Error("myzk.RegisterTemp() error(%v)", err)
		return conn, err
	}
	// watch and update
	go watchCometRoot(conn, Conf.ZookeeperCometPath, Conf.KetamaBase)
	rpc.InitMessage(conn, Conf.ZookeeperMessagePath, Conf.RPCRetry, Conf.RPCPing, Conf.KetamaBase)
	return conn, nil
}

// watchCometRoot monitoring all Comet nodes
func watchCometRoot(conn *zk.Conn, fpath string, vnode int) {
	for {
		nodes, watch, err := myzk.GetNodesW(conn, fpath)
		if err != nil {
			log.Error("myzk.GetNodesW() error(%v)", err)
			continue
		}
		tmp := make(map[string]*myrpc.CometNodeInfo)
		for _, node := range nodes {
			info, err := myrpc.GetNodesInfo(conn, node, fpath, vnode)
			if err != nil {
				log.Error("myrpc.GetNodesInfo() error(%v)", err)
				continue
			}
			tmp[node] = info
		}

		// handle nodes changed(eg:add or del)
		count := 0
		changed := false
		for _, node := range nodes {
			if _, ok := cometNodeInfoMap[node]; !ok {
				changed = true
				break
			}
			count++
		}
		cometNodeInfoMap = tmp
		log.Info("cometnode info :%d", len(cometNodeInfoMap))
		if changed {
			UserChannel.Migrate()
		} else {
			if count != len(nodes) {
				UserChannel.Migrate()
			}
		}

		// blocking wait node changed
		event := <-watch
		log.Info("zk path: \"%s\" receive a event %v", fpath, event)
	}
}
