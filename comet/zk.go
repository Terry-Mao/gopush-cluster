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
	"launchpad.net/gozk/zookeeper"
	"strings"
)

var (
	ErrNodeName = errors.New("zookeeper node name must not contain \",\"")
)

type ZK struct {
	conn *zookeeper.Conn
}

// newZookeeper dial zookeeper cluster.
func newZookeeper() (*ZK, error) {
	zk, session, err := zookeeper.Dial(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		Log.Error("zookeeper.Dial(\"%s\", %d) error(%v)", Conf.ZookeeperAddr, Conf.ZookeeperTimeout, err)
		return nil, err
	}
	go func() {
		for {
			event := <-session
			if event.State < zookeeper.STATE_CONNECTING {
				Log.Error("can't connect zookeeper, event: %v", event)
			} else if event.State == zookeeper.STATE_CONNECTING {
				Log.Warn("retry connect zookeeper, event: %v", event)
			} else {
				Log.Debug("succeed connect zookeeper, event: %v", event)
			}
		}
	}()
	return &ZK{conn: zk}, nil
}

// Create the persistence node in zookeeper
func (zk *ZK) create() error {
	// create zk root path
	tpath := ""
	for _, str := range strings.Split(Conf.ZookeeperPath, "/")[1:] {
		tpath += "/" + str
		Log.Debug("create zookeeper path:%s", tpath)
		_, err := zk.conn.Create(tpath, "", 0, zookeeper.WorldACL(zookeeper.PERM_ALL))
		if err != nil {
			if zookeeper.IsError(err, zookeeper.ZNODEEXISTS) {
				Log.Warn("zk.create(\"%s\") exists", tpath)
			} else {
				Log.Error("zk.create(\"%s\") error(%v)", tpath, err)
				return err
			}
		}
	}
	// create node path
	fpath := fmt.Sprintf("%s/%s", Conf.ZookeeperPath, Conf.ZookeeperNode)
	Log.Debug("create zookeeper path:%s", fpath)
	_, err := zk.conn.Create(fpath, "", 0, zookeeper.WorldACL(zookeeper.PERM_ALL))
	if err != nil {
		if zookeeper.IsError(err, zookeeper.ZNODEEXISTS) {
			Log.Warn("zk.Create(\"%s\") exists", fpath)
		} else {
			Log.Error("zk.Create(\"%s\") error(%v)", fpath, err)
			return err
		}
	}
	return nil
}

// register register a node in zookeeper, when comet exit the node will remove
func (zk *ZK) register() error {
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
	fpath := fmt.Sprintf("%s/%s/", Conf.ZookeeperPath, Conf.ZookeeperNode)
	tpath, err := zk.conn.Create(fpath, data, zookeeper.EPHEMERAL|zookeeper.SEQUENCE, zookeeper.WorldACL(zookeeper.PERM_ALL))
	if err != nil {
		Log.Error("zk.conn.Create(\"%s\", \"%s\", zookeeper.EPHEMERAL|zookeeper.SEQUENCE) error(%v)", fpath, data, err)
		return err
	}
	Log.Debug("create a zookeeper node:%s", tpath)
	return nil
}

// Close close zookeeper connection.
func (zk *ZK) Close() {
	Log.Info("zookeeper addr: \"%s\" close", Conf.ZookeeperAddr)
	if err := zk.conn.Close(); err != nil {
		Log.Error("zk.conn.Close() error(%v)", err)
	}
}

// InitZookeeper init the node path and register node in zookeeper.
func InitZookeeper() (*ZK, error) {
	// create a zk conn
	zk, err := newZookeeper()
	if err != nil {
		Log.Error("newZookeeper() error(%v)", err)
		return nil, err
	}
	// init zk path
	if err = zk.create(); err != nil {
		Log.Error("zk.create() error(%v)", err)
		return nil, err
	}
	// register zk node, dns,adminaddr
	if err = zk.register(); err != nil {
		Log.Error("zk.register() error(%v)", err)
		return nil, err
	}
	return zk, nil
}
