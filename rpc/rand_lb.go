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
	"errors"
	"fmt"
	"github.com/golang/glog"
	"math/rand"
	"net/rpc"
	"time"
)

const (
	randLBRetryCHLength = 10
)

var (
	ErrRandLBLength = errors.New("clients and addrs length not match")
	ErrRandLBAddr   = errors.New("clients map no addr key")
)

// random load balancing object
type RandLB struct {
	Clients map[string]*rpc.Client
	addrs   []string
	length  int
	exitCH  chan int
}

// NewRandLB new a random load balancing object.
func NewRandLB(clients map[string]*rpc.Client, addrs []string, service string, retry, ping time.Duration, check bool) (*RandLB, error) {
	length := len(clients)
	if length != len(addrs) {
		glog.Errorf("clients: %d, addrs: %d not equal", length, len(addrs))
		return nil, ErrRandLBLength
	}
	for _, addr := range addrs {
		if _, ok := clients[addr]; !ok {
			glog.Errorf("addr: \"%s\" not exist in clients map", addr)
			return nil, ErrRandLBAddr
		}
	}
	r := &RandLB{Clients: clients, addrs: addrs, length: length}
	if check && length > 0 {
		glog.Info("rpc ping start")
		r.ping(service, retry, ping)
	}
	return r, nil
}

// Get get a rpc client randomly.
func (r *RandLB) Get() *rpc.Client {
	if len(r.addrs) == 0 {
		return nil
	}
	addr := r.addrs[rand.Intn(r.length)]
	glog.V(1).Infof("rand hit rpc node: \"%s\"", addr)
	client, _ := r.Clients[addr]
	return client
}

// Stop stop the retry connect goroutine and ping goroutines.
func (r *RandLB) Stop() {
	if r.exitCH != nil {
		close(r.exitCH)
	}
	glog.Info("stop the randlb retry connect goroutine and ping goroutines")
}

// Destroy release the rpc.Client resource.
func (r *RandLB) Destroy() {
	r.Stop()
	for _, client := range r.Clients {
		if client != nil {
			if err := client.Close(); err != nil {
				glog.Errorf("client.Close() error(%v)", err)
			}
		}
	}
}

// ping do a ping, if failed then retry.
func (r *RandLB) ping(service string, retry, ping time.Duration) {
	method := fmt.Sprintf("%s.Ping", service)
	retryCH := make(chan string, randLBRetryCHLength)
	r.exitCH = make(chan int, 1)
	for _, addr := range r.addrs {
		// warn: closures problem
		go func(addr string) {
			glog.Infof("\"%s\" rpc ping goroutine start", addr)
			ret := 0
			for {
				select {
				case <-r.exitCH:
					glog.Infof("\"%s\" rpc ping goroutine exit", addr)
					return
				default:
				}
				// get client for ping
				client, _ := r.Clients[addr]
				if err := client.Call(method, 0, &ret); err != nil {
					// if failed send to chan reconnect, sleep
					client.Close()
					retryCH <- addr
					glog.Errorf("client.Call(\"%s\", 0, &ret) error(%v), retry", method, err)
					time.Sleep(retry)
					continue
				}
				// if ok, sleep
				glog.V(2).Infof("\"%s\": rpc ping ok", addr)
				time.Sleep(ping)
			}
		}(addr)
	}
	// rpc retry connect
	go func() {
		var retryAddr string
		glog.Info("rpc retry connect goroutine start")
		for {
			select {
			case retryAddr = <-retryCH:
			case <-r.exitCH:
				glog.Info("rpc retry connect goroutine exit")
				return
			}
			rpcTmp, err := rpc.Dial("tcp", retryAddr)
			if err != nil {
				glog.Errorf("rpc.Dial(\"tcp\", %s) error(%s)", retryAddr, err)
				continue
			}
			glog.Infof("rpc.Dial(\"tcp\", %s) retry succeed", retryAddr)
			// copy-on-write
			tmpClients := make(map[string]*rpc.Client, r.length)
			for addr, client := range r.Clients {
				tmpClients[addr] = client
			}
			tmpClients[retryAddr] = rpcTmp
			// atomic update clients
			r.Clients = tmpClients
		}
	}()
}
