package main

import (
	"errors"
	"launchpad.net/gozk/zookeeper"
	"strings"
	"time"
)

var (
	NodeNameErr = errors.New("zookeeper node name must not contain \",\"")
)

type ZK struct {
	conn *zookeeper.Conn
}

func NewZookeeper(addr string, timeout int) (*ZK, error) {
	zk, session, err := zookeeper.Dial(addr, time.Duration(timeout)*1e9)
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
func (zk *ZK) Create(path string, node string) error {
	// create zk root path
	tpath := ""
	for _, str := range strings.Split(path, "/")[1:] {
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
	Log.Debug("create zookeeper path:%s", fpath)
	fpath := path + "/" + node
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

// Register register a node in zookeeper, when comet exit the node will remove
func (zk *ZK) Register(path string, node string, val string) error {
	fpath := path + "/" + node + "/"
	tpath, err := zk.conn.Create(fpath, val, zookeeper.EPHEMERAL|zookeeper.SEQUENCE, zookeeper.WorldACL(zookeeper.PERM_ALL))
	if err != nil {
		Log.Error("zk.Create(\"%s\", \"%s\", zookeeper.EPHEMERAL|zookeeper.SEQUENCE) failed (%s)", path, val, err.Error())
		return err
	}

	Log.Debug("create a zookeeper node:%s", tpath)
	return nil
}

// InitZookeeper init the zk path and register node in zk
func InitZookeeper() error {
	// create a zk conn
	zk, err := NewZookeeper(Conf.ZookeeperAddr, Conf.ZookeeperTimeout)
	if err != nil {
		Log.Error("NewZookeeper() failed (%s)", err.Error())
		return err
	}

	// init zk path
	Log.Info("init zookeeper root path and node path")
	if err = zk.Create(Conf.ZookeeperPath, Conf.Node); err != nil {
		Log.Error("zk.Create(\"%s\") failed (%s)", Conf.ZookeeperPath, err.Error())
		return err
	}

	// register zk node
	Log.Info("register %s:%s in zookeeper:%s", Conf.Node, Conf.DNS, Conf.ZookeeperPath)
	if err = zk.Register(Conf.ZookeeperPath, Conf.Node, Conf.DNS); err != nil {
		Log.Error("zk.Register(\"%s\", \"%s\") failed (%s)", Conf.ZookeeperPath, Conf.DNS, err.Error())
		return err
	}

	return nil
}
