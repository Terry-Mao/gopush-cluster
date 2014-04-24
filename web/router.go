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
	"github.com/Terry-Mao/gopush-cluster/router"
	"github.com/golang/glog"
)

const (
	NetworkRouterCN = "CN" // china
)

var (
	routerCN *router.RouterCN
)

// Init network router
func InitRouter() error {
	switch Conf.Router {
	case NetworkRouterCN:
		r, err := router.InitCN(Conf.QQWryPath)
		if err != nil {
			glog.Errorf("init china network router failed(%v)", err)
			return err
		}
		routerCN = r
	default:
		// TODO:support more countries` network routers
		return fmt.Errorf("unknown network router:\"%s\", please check your configuration", Conf.Router)
	}

	return nil
}
