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
	"fmt"
	timeID "github.com/Terry-Mao/gopush-cluster/id"
	"launchpad.net/gozk/zookeeper"
	"sort"
	"strings"
)

// Public message id-creater
var PubMID *timeID.TimeID

// PubMIDLock public message mid lock, make sure that get the unique mid
func PubMIDLock() (bool, string, error) {
	prefix := "p"
	splitSign := "@"
	pathCreated, err := zk.Create(fmt.Sprintf("%s/%s%s", Conf.ZKPIDPath, prefix, splitSign),
		"0", zookeeper.EPHEMERAL|zookeeper.SEQUENCE, zookeeper.WorldACL(zookeeper.PERM_ALL))
	if err != nil {
		return false, pathCreated, fmt.Errorf("zk.Create(%s/%s%s) error(%v)", Conf.ZKPIDPath, prefix, splitSign, err)
	}

	for {
		childrens, stat, err := zk.Children(Conf.ZKPIDPath)
		if err != nil {
			return false, pathCreated, fmt.Errorf("zk.Children(%s) error(%v)", Conf.ZKPIDPath, err)
		}

		// If node isn`t exist
		if childrens == nil || stat == nil {
			return false, pathCreated, fmt.Errorf("node(%s) is not existent", Conf.ZKPIDPath)
		}

		var realQueue []string
		for _, children := range childrens {
			tmp := strings.Split(children, splitSign)
			if prefix == tmp[0] {
				realQueue = append(realQueue, children)
			}
		}

		// Sort sequence nodes
		sort.Strings(realQueue)

		tmp := strings.Split(pathCreated, "/")
		posReal := sort.StringSlice(realQueue).Search(tmp[len(tmp)-1])

		// If does not get lock
		if posReal > 0 {
			// Watch the last one
			watchPath := fmt.Sprintf("%s/%s", Conf.ZKPIDPath, realQueue[posReal-1])
			_, watch, err := zk.ExistsW(watchPath)
			if err != nil || zookeeper.IsError(err, zookeeper.ZNONODE) {
				return false, pathCreated, fmt.Errorf("zk.ExistsW(%s) error(%v) or no node", watchPath, err)
			}

			// Watch the lower node
			watchNode := <-watch
			switch watchNode.Type {
			case zookeeper.EVENT_DELETED:
			default:
				return false, pathCreated, fmt.Errorf("zookeeper watch errCode:%d", watchNode.Type)
			}

			return false, pathCreated, nil
		} else {
			return true, pathCreated, nil
		}
	}

	// Never get here
	return false, pathCreated, fmt.Errorf("never get here")
}

// PubMIDLockRelease release the public message id-lock
func PubMIDLockRelease(pathCreated string) error {
	if err := zk.Delete(pathCreated, -1); err != nil {
		return err
	}
	return nil
}
