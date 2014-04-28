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
	"fmt"
	"github.com/golang/glog"
	"net"
	"net/http"
	"time"
)

const (
	httpReadTimeout = 30 //seconds
)

// StartHTTP start listen http.
func StartHTTP() {
	// external
	httpServeMux := http.NewServeMux()
	httpServeMux.HandleFunc("/1/server/get", GetServer)
	httpServeMux.HandleFunc("/1/msg/get", GetOfflineMsg)
	httpServeMux.HandleFunc("/1/time/get", GetTime)
	// internal
	httpAdminServeMux := http.NewServeMux()
	httpAdminServeMux.HandleFunc("/admin/push", PushPrivate)
	httpAdminServeMux.HandleFunc("/admin/msg/clean", DelOfflineMessage)
	for _, bind := range Conf.Addr {
		glog.Infof("start http listen addr:\"%s\"", bind)
		go httpListen(httpServeMux, bind)
	}
	for _, bind := range Conf.AdminAddr {
		glog.Infof("start admin http listen addr:\"%s\"", bind)
		go httpListen(httpAdminServeMux, bind)
	}
}

func httpListen(mux *http.ServeMux, bind string) {
	server := &http.Server{Handler: mux, ReadTimeout: httpReadTimeout * time.Second}
	l, err := net.Listen("tcp", bind)
	if err != nil {
		glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
		panic(err)
	}
	if err := server.Serve(l); err != nil {
		glog.Errorf("server.Serve() error(%v)", err)
		panic(err)
	}
}

// retWrite marshal the result and write to client(get).
func retWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, callback string, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		glog.Errorf("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := ""
	if callback == "" {
		// Normal json
		dataStr = string(data)
	} else {
		// Jsonp
		dataStr = fmt.Sprintf("%s(%s)", callback, string(data))
	}
	if n, err := w.Write([]byte(dataStr)); err != nil {
		glog.Errorf("w.Write(\"%s\") error(%v)", dataStr, err)
	} else {
		glog.V(1).Infof("w.Write(\"%s\") write %d bytes", dataStr, n)
	}
	glog.Infof("req: \"%s\", res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}

// retPWrite marshal the result and write to client(post).
func retPWrite(w http.ResponseWriter, r *http.Request, res map[string]interface{}, body *string, start time.Time) {
	data, err := json.Marshal(res)
	if err != nil {
		glog.Errorf("json.Marshal(\"%v\") error(%v)", res, err)
		return
	}
	dataStr := string(data)
	if n, err := w.Write([]byte(dataStr)); err != nil {
		glog.Errorf("w.Write(\"%s\") error(%v)", dataStr, err)
	} else {
		glog.V(1).Infof("w.Write(\"%s\") write %d bytes", dataStr, n)
	}
	glog.Infof("req: \"%s\", post: \"%s\", res:\"%s\", ip:\"%s\", time:\"%fs\"", r.URL.String(), *body, dataStr, r.RemoteAddr, time.Now().Sub(start).Seconds())
}
