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
	"fmt"
	"github.com/golang/glog"
	"github.com/samuel/go-zookeeper/zk"
	"strings"
)

func InitZK() (*zk.Conn, error) {
	// connect to zookeeper, get event from chan in goroutine(log)
	conn, session, err := zk.Connect(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		glog.Errorf("zk.Connect(\"%v\", %d) error(%v)", Conf.ZookeeperAddr, Conf.ZookeeperTimeout, err)
		return nil, err
	}
	go func() {
		for {
			event := <-session
			glog.Infof("zookeeper get a event: %s", event.State.String())
		}
	}()
	// create zk root path
	tpath := ""
	for _, str := range strings.Split(Conf.ZookeeperPath, "/")[1:] {
		tpath += "/" + str
		glog.V(1).Infof("create zookeeper path:%s", tpath)
		_, err = conn.Create(tpath, []byte(""), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			if err == zk.ErrNodeExists {
				glog.Warningf("zk.create(\"%s\") exists", tpath)
			} else {
				glog.Errorf("zk.create(\"%s\") error(%v)", tpath, err)
				return nil, err
			}
		}
	}
	// create node path
	fpath := fmt.Sprintf("%s/%s", Conf.ZookeeperPath, Conf.ZookeeperNode)
	glog.V(1).Infof("create zookeeper path:%s", fpath)
	_, err = conn.Create(fpath, []byte(""), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists {
			glog.Warningf("zk.Create(\"%s\") exists", fpath)
		} else {
			glog.Errorf("zk.Create(\"%s\") error(%v)", fpath, err)
			return nil, err
		}
	}
	// tcp, websocket and rpc bind address store in the zk
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
	tpath, err = conn.Create(fpath+"/", []byte(data), zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		glog.Errorf("conn.Create(\"%s\", \"%s\", zk.FlagEphemeral|zk.FlagSequence) error(%v)", fpath, data, err)
		return nil, err
	}
	glog.V(1).Infof("create a zookeeper node:%s", tpath)
	// watch self
	go func() {
		for {
			glog.Infof("zk path: \"%s\" set a watch", tpath)
			exist, _, watch, err := conn.ExistsW(tpath)
			if err != nil {
				glog.Errorf("zk.ExistsW(\"%s\") error(%v)", tpath, err)
				glog.Warningf("zk path: \"%s\" set watch failed, comet kill itself", tpath)
				KillSelf()
				return
			}
			if !exist {
				glog.Warningf("zk path: \"%s\" not exist, comet kill itself", tpath)
				KillSelf()
				return
			}
			event := <-watch
			glog.Infof("zk path: \"%s\" receive a event %v", tpath, event)
		}
	}()
	return conn, nil
}
