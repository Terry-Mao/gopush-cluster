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
	log "code.google.com/p/log4go"
	"encoding/json"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
		res["ret"] = InternalErr
		log.Error("ioutil.ReadAll() failed (%v)", err)
		return
	}
	body = string(bodyBytes)
	params := r.URL.Query()
	key := params.Get("key")
	expire, err := strconv.ParseUint(params.Get("expire"), 10, 32)
	if err != nil {
		res["ret"] = ParamErr
		log.Error("strconv.ParseUint(\"%s\", 10, 32) error(%v)", params.Get("expire"), err)
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
		log.Error("json.RawMessage(\"%s\").MarshalJSON() error(%v)", body, err)
		return
	}
	args := &myrpc.CometPushPrivateArgs{Msg: json.RawMessage(msg), Expire: uint(expire), Key: key}
	ret := 0
	if err := client.Call(myrpc.CometServicePushPrivate, args, &ret); err != nil {
		log.Error("client.Call(\"%s\", \"%s\", &ret) error(%v)", myrpc.CometServicePushPrivate, args.Key, err)
		res["ret"] = InternalErr
		return
	}
	return
}

// PushMultiPrivate handle for push multiple private messages.
// because of it`s going asynchronously in this method, so it won`t return an InternalErr to caller.
func PushMultiPrivate(w http.ResponseWriter, r *http.Request) {
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
		res["ret"] = InternalErr
		log.Error("ioutil.ReadAll() failed (%v)", err)
		return
	}
	msgBytes, keys, ret := parseMultiPrivate(bodyBytes)
	if ret != OK {
		res["ret"] = ret
		return
	}
	rm := json.RawMessage(msgBytes)
	msg, err := rm.MarshalJSON()
	if err != nil {
		res["ret"] = ParamErr
		log.Error("json.RawMessage(\"%s\").MarshalJSON() error(%v)", string(msg), err)
		return
	}

	params := r.URL.Query()
	expire, err := strconv.ParseUint(params.Get("expire"), 10, 32)
	if err != nil {
		res["ret"] = ParamErr
		log.Error("strconv.ParseUint(\"%s\", 10, 32) error(%v)", params.Get("expire"), err)
		return
	}
	// match nodes
	nodes := map[*myrpc.CometNodeInfo]*[]string{}
	for i := 0; i < len(keys); i++ {
		node := myrpc.GetComet(keys[i])
		if node == nil || node.CometRPC == nil {
			res["ret"] = NotFoundServer
			return
		}
		keysTmp, ok := nodes[node]
		if ok {
			*keysTmp = append(*keysTmp, keys[i])
		} else {
			nodes[node] = &[]string{keys[i]}
		}
	}
	var fKeys []string
	for cometInfo, ks := range nodes {
		client := cometInfo.CometRPC.Get()
		if client == nil {
			log.Error("cannot get comet rpc client")
			for i := 0; i < len(*ks); i++ {
				fKeys = append(fKeys, (*ks)[i])
			}
			continue
		}
		args := &myrpc.CometPushPrivatesArgs{Msg: json.RawMessage(msg), Expire: uint(expire), Keys: *ks}
		resp := myrpc.CometPushPrivatesResp{}
		if err := client.Call(myrpc.CometServicePushPrivates, args, &resp); err != nil {
			log.Error("client.Call(\"%s\", \"%v\", &ret) error(%v)", myrpc.CometServicePushPrivates, args.Keys, err)
			for i := 0; i < len(*ks); i++ {
				fKeys = append(fKeys, (*ks)[i])
			}
			continue
		}

		for i := 0; i < len(resp.FKeys); i++ {
			fKeys = append(fKeys, resp.FKeys[i])
		}
	}
	res["ret"] = OK
	if len(fKeys) != 0 {
		res["data"] = map[string]interface{}{"fk": fKeys}
	}
	return
}

// parseMultiPrivate gets keys and msg what need to push.
// body eg: {"m":"push messages json string","k":"key1,key2,key3"}, must be a json.
// field k join through ','.
func parseMultiPrivate(body []byte) (msg []byte, keys []string, ret int) {
	b := map[string]string{}
	if err := json.Unmarshal(body, &b); err != nil {
		log.Error("json.Unmarshal(\"%s\") error(%v)", string(body), err)
		ret = ParamErr
		return
	}
	//message
	m := b["m"]
	if len(m) == 0 {
		ret = ParamErr
		return
	}
	//keys
	k := b["k"]
	if k == "" {
		ret = ParamErr
		return
	}
	keys = strings.Split(k, ",")
	msg = []byte(m)
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
		log.Error("ioutil.ReadAll() failed (%v)", err)
		return
	}
	body = string(bodyBytes)
	params, err := url.ParseQuery(body)
	if err != nil {
		log.Error("url.ParseQuery(\"%s\") error(%v)", body, err)
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
		log.Warn("user_key: \"%s\" can't not find message rpc node", key)
		res["ret"] = InternalErr
		return
	}
	ret := 0
	if err := client.Call(myrpc.MessageServiceDelPrivate, key, &ret); err != nil {
		log.Error("client.Call(\"%s\", \"%s\", &ret) error(%v)", myrpc.MessageServiceDelPrivate, key, err)
		res["ret"] = InternalErr
		return
	}
	return
}
