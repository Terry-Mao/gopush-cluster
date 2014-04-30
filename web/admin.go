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
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PushPrivate handle for push private message.
func PushPrivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	body := ""
	res := map[string]interface{}{"ret": OK}
	defer retPWrite(w, r, res, &body, time.Now())
	// param
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)
	params := r.URL.Query()
	key := params.Get("key")
	expireStr := params.Get("expire")
	if key == "" {
		res["ret"] = ParamErr
		return
	}
	expire, err := strconv.ParseUint(expireStr, 10, 32)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("strconv.ParseUint(\"%s\", 10, 32) error(%v)", expireStr, err)
		return
	}
	node := myrpc.GetComet(key)
	if node == nil || node.CometRPC == nil {
		res["ret"] = NotFoundServer
		return
	}
	client := node.CometRPC.Get()
	if client == nil {
		res["ret"] = NotFoundServer
		return
	}
	rm := json.RawMessage(bodyBytes)
	msg, err := rm.MarshalJSON()
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("json.RawMessage(\"%s\").MarshalJSON() error(%v)", body, err)
		return
	}
	args := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msg), Expire: uint(expire), Key: key}
	ret := 0
	if err := client.Call(myrpc.CometServicePushPrivate, args, &ret); err != nil {
		glog.Errorf("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivate, args, err)
		res["ret"] = InternalErr
		return
	}
	return
}

// DelPrivate handle for push private message.
func DelPrivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", 405)
		return
	}
	body := ""
	res := map[string]interface{}{"ret": OK}
	defer retPWrite(w, r, res, &body, time.Now())
	// param
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		res["ret"] = ParamErr
		glog.Errorf("ioutil.ReadAll() failed (%s)", err.Error())
		return
	}
	body = string(bodyBytes)
	params, err := url.ParseQuery(body)
	if err != nil {
		glog.Errorf("url.ParseQuery(\"%s\") error(%v)", body, err)
		res["ret"] = ParamErr
		return
	}
	key := params.Get("key")
	if key == "" {
		res["ret"] = ParamErr
		return
	}
	client := myrpc.MessageRPC.Get()
	if client == nil {
		res["ret"] = InternalErr
		return
	}
	ret := 0
	if err := client.Call(myrpc.MessageServiceDelPrivate, key, &ret); err != nil {
		glog.Errorf("client.Call(\"%s\", \"%s\", &ret) error(%v)", myrpc.MessageServiceDelPrivate, key, err)
		res["ret"] = InternalErr
		return
	}
	return
}
