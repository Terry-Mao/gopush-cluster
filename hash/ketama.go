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

package hash

import (
	//	"crypto/md5"
	"fmt"
	"sort"
)

// Convenience types for common cases
// UIntSlice attaches the methods of Interface to []uint, sorting in increasing order.
type UIntSlice []uint

func (p UIntSlice) Len() int           { return len(p) }
func (p UIntSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p UIntSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type Ketama struct {
	node         int             // phsical node number
	vnode        int             // virual node number
	nodes        []uint          // nodes
	nodesMapping map[uint]string // nodes maping
}

// New create a ketama consistent hashing struct
func NewKetama(node, vnode int) *Ketama {
	ketama := &Ketama{}
	ketama.node = node
	ketama.vnode = vnode
	ketama.nodes = []uint{}
	ketama.nodesMapping = map[uint]string{}

	ketama.initCircle()

	return ketama
}

// New create a ketama consistent hashing struct use exist node slice
func NewKetama2(node []string, vnode int) *Ketama {
	ketama := &Ketama{}
	ketama.node = len(node)
	ketama.vnode = vnode
	ketama.nodes = []uint{}
	ketama.nodesMapping = map[uint]string{}

	ketama.initCircle2(node)

	return ketama
}

// init consistent hashing circle
func (k *Ketama) initCircle() {
	h := NewMurmur3C()
	for idx := 1; idx < k.node+1; idx++ {
		node := fmt.Sprintf("node%d", idx)
		for i := 0; i < k.vnode; i++ {
			vnode := fmt.Sprintf("%s#%d", node, i)
			h.Write([]byte(vnode))
			vpos := uint(h.Sum32())
			k.nodes = append(k.nodes, vpos)
			k.nodesMapping[vpos] = node
			h.Reset()
		}
	}

	sort.Sort(UIntSlice(k.nodes))
}

// init consistent hashing circle
func (k *Ketama) initCircle2(node []string) {
	h := NewMurmur3C()
	for _, str := range node {
		for i := 0; i < k.vnode; i++ {
			vnode := fmt.Sprintf("%s#%d", str, i)
			h.Write([]byte(vnode))
			vpos := uint(h.Sum32())
			k.nodes = append(k.nodes, vpos)
			k.nodesMapping[vpos] = str
			h.Reset()
		}
	}

	sort.Sort(UIntSlice(k.nodes))
}

// Node get a consistent hashing node by key
func (k *Ketama) Node(key string) string {
	if len(k.nodes) == 0 {
		return ""
	}
	h := NewMurmur3C()
	h.Write([]byte(key))
	idx := searchLeft(k.nodes, uint(h.Sum32()))
	pos := k.nodes[0]
	if idx != len(k.nodes) {
		pos = k.nodes[idx]
	}

	return k.nodesMapping[pos]
}

// search the slice left-most value
func searchLeft(a []uint, x uint) int {
	lo := 0
	hi := len(a)
	for lo < hi {
		mid := (lo + hi) / 2
		if a[mid] < x {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	return lo
}
