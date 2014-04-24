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
	"encoding/json"
	"github.com/golang/glog"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	// server
	startTime int64 // process start unixnano
	// channel
	ChStat = &ChannelStat{}
	// message
	MsgStat = &MessageStat{}
	// connection
	ConnStat = &ConnectionStat{}
)

// Channel stat info
type ChannelStat struct {
	Access uint64 // total access count
	Create uint64 // total create count
	Delete uint64 // total delete count
}

func (s *ChannelStat) IncrAccess() {
	atomic.AddUint64(&s.Access, 1)
}

func (s *ChannelStat) IncrCreate() {
	atomic.AddUint64(&s.Create, 1)
}

func (s *ChannelStat) IncrDelete() {
	atomic.AddUint64(&s.Delete, 1)
}

// Stat get the channle stat info
func (s *ChannelStat) Stat() []byte {
	res := map[string]interface{}{}
	res["access"] = s.Access
	res["create"] = s.Create
	res["delete"] = s.Delete
	res["current"] = UserChannel.Count()
	return jsonRes(res)
}

// Message stat info
type MessageStat struct {
	Succeed uint64 // total push message succeed count
	Failed  uint64 // total push message failed count
}

func (s *MessageStat) IncrSucceed(delta uint64) {
	atomic.AddUint64(&s.Succeed, delta)
}

func (s *MessageStat) IncrFailed(delta uint64) {
	atomic.AddUint64(&s.Failed, delta)
}

// Stat get the message stat info
func (s *MessageStat) Stat() []byte {
	res := map[string]interface{}{}
	res["succeed"] = s.Succeed
	res["failed"] = s.Failed
	res["total"] = s.Succeed + s.Failed
	return jsonRes(res)
}

// Connection stat info
type ConnectionStat struct {
	Add    uint64 // total add connection count
	Remove uint64 // total remove connection count
}

func (s *ConnectionStat) IncrAdd() {
	atomic.AddUint64(&s.Add, 1)
}

func (s *ConnectionStat) IncrRemove() {
	atomic.AddUint64(&s.Remove, 1)
}

// Stat get the connection stat info
func (s *ConnectionStat) Stat() []byte {
	res := map[string]interface{}{}
	res["add"] = s.Add
	res["remove"] = s.Remove
	res["current"] = s.Add - s.Remove
	return jsonRes(res)
}

func statListen(bind string) {
	httpServeMux := http.NewServeMux()
	httpServeMux.HandleFunc("/stat", StatHandle)
	if err := http.ListenAndServe(bind, httpServeMux); err != nil {
		glog.Errorf("http.ListenAdServe(\"%s\") error(%v)", bind, err)
		panic(err)
	}
}

// start stats, called at process start
func StartStats() {
	startTime = time.Now().UnixNano()
	for _, bind := range Conf.StatBind {
		glog.Infof("start stat listen addr:\"%s\"", bind)
		go statListen(bind)
	}
}

// memory stats
func MemStats() []byte {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	// general
	res := map[string]interface{}{}
	res["alloc"] = m.Alloc
	res["total_alloc"] = m.TotalAlloc
	res["sys"] = m.Sys
	res["lookups"] = m.Lookups
	res["mallocs"] = m.Mallocs
	res["frees"] = m.Frees
	// heap
	res["heap_alloc"] = m.HeapAlloc
	res["heap_sys"] = m.HeapSys
	res["heap_idle"] = m.HeapIdle
	res["heap_inuse"] = m.HeapInuse
	res["heap_released"] = m.HeapReleased
	res["heap_objects"] = m.HeapObjects
	// low-level fixed-size struct alloctor
	res["stack_inuse"] = m.StackInuse
	res["stack_sys"] = m.StackSys
	res["mspan_inuse"] = m.MSpanInuse
	res["mspan_sys"] = m.MSpanSys
	res["mcache_inuse"] = m.MCacheInuse
	res["mcache_sys"] = m.MCacheSys
	res["buckhash_sys"] = m.BuckHashSys
	// GC
	res["next_gc"] = m.NextGC
	res["last_gc"] = m.LastGC
	res["pause_total_ns"] = m.PauseTotalNs
	res["pause_ns"] = m.PauseNs
	res["num_gc"] = m.NumGC
	res["enable_gc"] = m.EnableGC
	res["debug_gc"] = m.DebugGC
	res["by_size"] = m.BySize
	return jsonRes(res)
}

// golang stats
func GoStats() []byte {
	res := map[string]interface{}{}
	res["compiler"] = runtime.Compiler
	res["arch"] = runtime.GOARCH
	res["os"] = runtime.GOOS
	res["max_procs"] = runtime.GOMAXPROCS(-1)
	res["root"] = runtime.GOROOT()
	res["cgo_call"] = runtime.NumCgoCall()
	res["goroutine_num"] = runtime.NumGoroutine()
	res["version"] = runtime.Version()
	return jsonRes(res)
}

// server stats
func ServerStats() []byte {
	res := map[string]interface{}{}
	res["uptime"] = time.Now().UnixNano() - startTime
	hostname, _ := os.Hostname()
	res["hostname"] = hostname
	wd, _ := os.Getwd()
	res["wd"] = wd
	res["ppid"] = os.Getppid()
	res["pid"] = os.Getpid()
	res["pagesize"] = os.Getpagesize()
	if usr, err := user.Current(); err != nil {
		glog.Errorf("user.Current() error(%v)", err)
		res["group"] = ""
		res["user"] = ""
	} else {
		res["group"] = usr.Gid
		res["user"] = usr.Uid
	}
	return jsonRes(res)
}

// configuration info
func ConfigInfo() []byte {
	byteJson, err := json.MarshalIndent(Conf, "", "    ")
	if err != nil {
		glog.Errorf("json.MarshalIndent(\"%v\", \"\", \"    \") error(%v)", Conf, err)
		return nil
	}
	return byteJson
}

// jsonRes format the output
func jsonRes(res map[string]interface{}) []byte {
	byteJson, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		glog.Errorf("json.MarshalIndent(\"%v\", \"\", \"    \") error(%v)", res, err)
		return nil
	}
	return byteJson
}

func ChInfoStat(key string) []byte {
	res := map[string]interface{}{}
	if ch, err := UserChannel.Get(key, false); err == nil {
		if sch, ok := ch.(*SeqChannel); ok {
			res["channel"] = map[string]interface{}{"conn": sch.conn.Len()}
		} else {
			return nil
		}
	} else {
		return nil
	}
	return jsonRes(res)
}

// StatHandle get stat info by http
func StatHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	params := r.URL.Query()
	types := params.Get("type")
	res := []byte{}
	switch types {
	case "memory":
		res = MemStats()
	case "server":
		res = ServerStats()
	case "golang":
		res = GoStats()
	case "config":
		res = ConfigInfo()
	case "channel":
		key := params.Get("key")
		if key == "" {
			res = ChStat.Stat()
		} else {
			res = ChInfoStat(key)
		}
	case "message":
		res = MsgStat.Stat()
	case "connection":
		res = ConnStat.Stat()
	default:
		http.Error(w, "Not Found", 404)
	}
	if res != nil {
		if _, err := w.Write(res); err != nil {
			glog.Errorf("w.Write(\"%s\") error(%v)", string(res), err)
		}
	}
}
