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
	"github.com/Terry-Mao/gopush-cluster/ketama"
	"math/rand"
	"net/rpc"
	"strconv"
	"time"
)

const (
	randLBRetryCHLength = 10
)

var (
	ErrRandLBLength = errors.New("clients and addrs length not match")
	ErrRandLBAddr   = errors.New("clients map no addr key")
)

type RPCClient struct {
	Client *rpc.Client
	Addr   string
	Weight int
}

// random load balancing object
type RandLB struct {
	Clients map[string]*RPCClient
	ring    *ketama.HashRing
	length  int
	exitCH  chan int
}

// NewRandLB new a random load balancing object.
func NewRandLB(clients map[string]*RPCClient, service string, retry, ping time.Duration, vnode int, check bool) (*RandLB, error) {
	ring := ketama.NewRing(vnode)
	for _, client := range clients {
		ring.AddNode(client.Addr, client.Weight)
	}
	ring.Bake()
	length := len(clients)
	r := &RandLB{Clients: clients, ring: ring, length: length}
	if check && length > 0 {
		log.Info("rpc ping start")
		r.ping(service, retry, ping)
	}

	return r, nil
}

// Get get a rpc client randomly.
func (r *RandLB) Get() *rpc.Client {
	if len(r.Clients) == 0 {
		return nil
	}

	addr := r.ring.Hash(strconv.FormatInt(rand.Int63n(time.Now().UnixNano()), 10))
	log.Debug("rand hit rpc node: \"%s\"", addr)

	return r.Clients[addr].Client
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
		go func(client *RPCClient) {
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
			tmpClients := make(map[string]*RPCClient, r.length)
			for addr, client := range r.Clients {
				tmpClients[addr] = client
				if client.Addr == retryAddr {
					client.Client = rpcTmp
				}
			}
			// atomic update clients
			r.Clients = tmpClients
		}
	}()
}
