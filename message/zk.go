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
	myzk "github.com/Terry-Mao/gopush-cluster/zk"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
)

// InitZK create zookeeper root path, and register a temp node.
func InitZK() (*zk.Conn, error) {
	conn, err := myzk.Connect(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		glog.Errorf("zk.Connect() error(%v)", err)
		return nil, err
	}
	if err = myzk.Create(conn, Conf.ZookeeperPath); err != nil {
		glog.Errorf("zk.Create() error(%v)", err)
		return conn, err
	}
	data := strings.Join(Conf.RPCBind, ",")
	glog.V(1).Infof("zk data: \"%s\"", data)
	// tcp, websocket and rpc bind address store in the zk
	if err = myzk.RegisterTemp(conn, Conf.ZookeeperPath, data); err != nil {
		glog.Errorf("zk.RegisterTemp() error(%v)", err)
		return conn, err
	}
	return conn, nil
}
