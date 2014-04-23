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

package router

import (
	"fmt"
	"github.com/thinkboy/go-qqwry"
	"strings"
)

const (
	CTCC = "电信"
	CMCC = "移动"
	CUCC = "联通"
	ENET = "教育网"
)

type RouterCN struct {
	qqWryReader *qqwry.QQWry
}

// Init china network router
func InitCN(path string) (*RouterCN, error) {
	qqWryRead, err := qqwry.NewQQWry(path)
	if err != nil {
		return nil, fmt.Errorf("load QQWry.dat error : %v", err)
	}

	return &RouterCN{qqWryReader: qqWryRead}, nil
}

// Select the best ip
func (r *RouterCN) SelectBest(remoteAddr string, ips []string) string {
	if len(ips) == 0 {
		return ""
	}

	ipRemote := strings.Split(remoteAddr, ":")
	cc := r.netName(ipRemote[0])
	if cc == "" {
		return ""
	}

	for i := 0; i < len(ips); i++ {
		ip := strings.Split(ips[i], ":")
		if cc == r.netName(ip[0]) {
			return ips[i]
		}
	}

	return ""
}

func (r *RouterCN) netName(ip string) string {
	cc := ""
	_, area := r.qqWryReader.QueryIP(ip)
	if -1 != strings.Index(area, CTCC) {
		cc = CTCC
	} else if -1 != strings.Index(area, CMCC) {
		cc = CMCC
	} else if -1 != strings.Index(area, CUCC) {
		cc = CUCC
	} else if -1 != strings.Index(area, ENET) {
		cc = ENET
	}

	return cc
}
