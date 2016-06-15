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
	log "github.com/alecthomas/log4go"
	"encoding/json"
	"github.com/Terry-Mao/gopush-cluster/rpc"
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/samuel/go-zookeeper/zk"
	"path"
	"time"
)

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
	if err = myzk.Create(conn, fpath); err != nil {
		log.Error("myzk.Create(conn,\"%s\",\"\") error(%v)", fpath, err)
		return conn, err
	}
	// comet tcp, websocket and rpc bind address store in the zk
	nodeInfo := &rpc.CometNodeInfo{}
	nodeInfo.RpcAddr = Conf.RPCBind
	nodeInfo.TcpAddr = Conf.TCPBind
	nodeInfo.WsAddr = Conf.WebsocketBind
	nodeInfo.Weight = Conf.ZookeeperCometWeight
	data, err := json.Marshal(nodeInfo)
	if err != nil {
		log.Error("json.Marshal() error(%v)", err)
		return conn, err
	}
	log.Debug("myzk node:\"%s\" registe data: \"%s\"", fpath, string(data))
	if err = myzk.RegisterTemp(conn, fpath, data); err != nil {
		log.Error("myzk.RegisterTemp() error(%v)", err)
		return conn, err
	}
	// watch and update
	rpc.InitMessage(conn, Conf.ZookeeperMessagePath, Conf.RPCRetry, Conf.RPCPing)
	return conn, nil
}
