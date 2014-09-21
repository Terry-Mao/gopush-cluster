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

package rpc

import (
	log "code.google.com/p/log4go"
	"errors"
	"fmt"
	"math/rand"
	"net/rpc"
	"sort"
	"time"
)

const (
	randLBRetryCHLength = 10
)

var (
	ErrRandLBLength = errors.New("clients and addrs length not match")
	ErrRandLBAddr   = errors.New("clients map no addr key")
)

// WeightRpc is a rand weight rpc struct.
type WeightRpc struct {
	Client *rpc.Client
	Addr   string
	Weight int
}

type byWeight []*WeightRpc

// Len is part of sort.Interface.
func (r byWeight) Len() int {
	return len(r)
}

// Swap is part of sort.Interface.
func (r byWeight) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Less is part of sort.Interface.
func (r byWeight) Less(i, j int) bool {
	return r[i].Weight < r[j].Weight
}

// random load balancing object
type RandLB struct {
	Clients map[string]*WeightRpc
	s       []*WeightRpc
	p       []float64
	exitCH  chan int
}

// NewRandLB new a random load balancing object.
func NewRandLB(clients map[string]*WeightRpc, service string, retry, ping time.Duration, vnode int, check bool) (*RandLB, error) {
	r := &RandLB{Clients: clients}
	r.initWeightRand()
	if check && len(clients) > 0 {
		log.Info("rpc ping start")
		r.ping(service, retry, ping)
	}
	return r, nil
}

// initRand init the rpc weight rand.
func (r *RandLB) initWeightRand() {
	if len(r.Clients) == 0 {
		return
	}
	total := 0.0
	s := []*WeightRpc{}
	for _, v := range r.Clients {
		s = append(s, v)
		total += float64(v.Weight)
	}
	sort.Sort(byWeight(s))
	p := []float64{}
	ratio := 0.0
	for i := 0; i < len(s)-1; i++ {
		ratio += float64(s[i].Weight) / total
		p = append(p, ratio)
	}
	p = append(p, float64(1))
	r.p = p
	r.s = s
}

// updateWeightRand update the rpc.Client by retryAddr.
func (r *RandLB) updateWeightRand(retryAddr string, rpcTmp *rpc.Client) {
	for i, c := range r.s {
		if c.Addr == retryAddr {
			r.s[i].Client = rpcTmp
			break
		}
	}
}

// Get get a rpc client randomly.
func (r *RandLB) Get() *rpc.Client {
	l := len(r.Clients)
	if l == 0 {
		return nil
	} else if l == 1 {
		return r.s[0].Client
	}
	return r.s[sort.Search(len(r.p), func(i int) bool { return r.p[i] >= rand.Float64() })].Client
}

// Stop stop the retry connect goroutine and ping goroutines.
func (r *RandLB) Stop() {
	if r.exitCH != nil {
		close(r.exitCH)
	}
	log.Info("stop the randlb retry connect goroutine and ping goroutines")
}

// Destroy release the rpc.Client resource.
func (r *RandLB) Destroy() {
	r.Stop()
	for _, client := range r.Clients {
		if client != nil {
			if err := client.Client.Close(); err != nil {
				log.Error("client.Close() error(%v)", err)
			}
		}
	}
}

// ping do a ping, if failed then retry.
func (r *RandLB) ping(service string, retry, ping time.Duration) {
	method := fmt.Sprintf("%s.Ping", service)
	retryCH := make(chan string, randLBRetryCHLength)
	r.exitCH = make(chan int, 1)
	for _, client := range r.Clients {
		// warn: closures problem
		go func(client *WeightRpc) {
			log.Info("\"%s\" rpc ping goroutine start", client.Addr)
			ret := 0
			for {
				select {
				case <-r.exitCH:
					log.Info("\"%s\" rpc ping goroutine exit", client.Addr)
					return
				default:
				}
				// get client for ping
				if err := client.Client.Call(method, 0, &ret); err != nil {
					// if failed send to chan reconnect, sleep
					client.Client.Close()
					retryCH <- client.Addr
					log.Error("client.Call(\"%s\", 0, &ret) error(%v), retry", method, err)
					time.Sleep(retry)
					continue
				}
				// if ok, sleep
				log.Debug("\"%s\": rpc ping ok", client.Addr)
				time.Sleep(ping)
			}
		}(client)
	}
	// rpc retry connect
	go func() {
		var retryAddr string
		log.Info("rpc retry connect goroutine start")
		for {
			select {
			case retryAddr = <-retryCH:
			case <-r.exitCH:
				log.Info("rpc retry connect goroutine exit")
				return
			}
			rpcTmp, err := rpc.Dial("tcp", retryAddr)
			if err != nil {
				log.Error("rpc.Dial(\"tcp\", %s) error(%s)", retryAddr, err)
				continue
			}
			log.Info("rpc.Dial(\"tcp\", %s) retry succeed", retryAddr)
			// copy-on-write
			tmpClients := make(map[string]*WeightRpc, len(r.Clients))
			for addr, client := range r.Clients {
				tmpClients[addr] = client
				if client.Addr == retryAddr {
					client.Client = rpcTmp
				}
			}
			// atomic update clients
			r.Clients = tmpClients
			// update rand s.
			r.updateWeightRand(retryAddr, rpcTmp)
		}
	}()
}
