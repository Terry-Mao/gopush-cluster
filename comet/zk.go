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

func newZookeeper() (*ZK, error) {
	zk, session, err := zookeeper.Dial(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
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
				Log.Warn("zk.Create(\"%s\") exists", tpath)
			} else {
				Log.Error("zk.Create(\"%s\") failed (%s)", tpath, err.Error())
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
			Log.Error("zk.Create(\"%s\") failed (%s)", fpath, err.Error())
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
		Log.Error("zk.Create(\"%s\", \"%s\", zookeeper.EPHEMERAL|zookeeper.SEQUENCE) failed (%s)", fpath, data, err.Error())
		return err
	}
	Log.Debug("create a zookeeper node:%s", tpath)
	return nil
}

// Close close zookeeper connection.
func (zk *ZK) Close() {
	if err := zk.conn.Close(); err != nil {
		Log.Error("zk.conn.Close() failed (%s)", err.Error())
	}
}

// InitZookeeper init the node path and register node in zookeeper.
func InitZookeeper() (*ZK, error) {
	// create a zk conn
	zk, err := newZookeeper()
	if err != nil {
		Log.Error("newZookeeper() failed (%s)", err.Error())
		return nil, err
	}
	// init zk path
	if err = zk.create(); err != nil {
		Log.Error("zk.Create() failed (%s)", err.Error())
		return nil, err
	}
	// register zk node, dns,adminaddr
	if err = zk.register(); err != nil {
		Log.Error("zk.register() failed (%s)", err.Error())
		return nil, err
	}
	return zk, nil
}
