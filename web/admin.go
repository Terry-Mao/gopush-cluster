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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// AdminPush handle for push private message
func AdminPushPrivate(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final response operation
	var bodyStr string
	defer func(body *string) {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		data, _ := json.Marshal(result)

		io.WriteString(rw, string(data))

		glog.Infof("request:push_private, request_url:\"%s\", request_body:\"%s\", ret:\"%d\"", r.URL.String(), *body, ret)
	}(&bodyStr)

	// Get params
	param := r.URL.Query()
	key := param.Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	groupId, err := strconv.Atoi(param.Get("gid"))
	if err != nil {
		ret = ParamErr
		return
	}

	expire, err := strconv.Atoi(param.Get("expire"))
	if err != nil {
		ret = ParamErr
		return
	}

	// Get message
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll() quest_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}
	bodyStr = string(body)

	// Match a push-server with the value computed through ketama algorithm
	svrInfo := FindNode(key)
	if svrInfo == nil || svrInfo.PubRPC == nil {
		glog.Errorf("no node for key: \"%s\"", key)
		ret = NoNodeErr
		return
	}

	// RPC call publish interface
	args := &myrpc.ChannelPushPrivateArgs{GroupId: uint(groupId), Msg: json.RawMessage(bodyStr), Expire: uint(expire), Key: key}
	if err := svrInfo.PubRPC.Call("ChannelRPC.PushPrivate", args, &ret); err != nil {
		glog.Errorf("RPC.Call(\"ChannelRPC.PushPrivate\") server:\"%v\" error(%v)", svrInfo.Addr, err)
		ret = InternalErr
		return
	}

	return
}

// AdminPushPub handle for push public message
func AdminPushPublic(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final response operation
	var bodyStr string
	defer func(body *string) {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		data, _ := json.Marshal(result)

		io.WriteString(rw, string(data))

		glog.Infof("request:push_public, request_url:\"%s\", request_body:\"%s\", ret:\"%d\"", r.URL.String(), *body, ret)
	}(&bodyStr)

	// Get params
	param := r.URL.Query()
	expire, err := strconv.ParseInt(param.Get("expire"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll() request_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}
	bodyStr = string(body)

	/*
		// Lock here, make sure that get the unique mid
		lockYes, pathCreated, err := PubMIDLock()
		if pathCreated != "" {
			defer PubMIDLockRelease(pathCreated)
		}
		if err != nil || lockYes == false {
			glog.Errorf("PubMIDLock error(%v)", err)
			ret = InternalErr
			return
		}
		mid := PubMID.ID()
	*/
	mid := int64(0)

	// Save public message
	expire = time.Now().Add(time.Duration(expire) * time.Second).Unix()
	reply, err := MessageRPCSavePub(string(body), mid, expire)
	// Message save failed
	if reply != OK {
		glog.Errorf("RPC.Call(\"MessageRPC.SavePub\") error (ret:\"%d\")", reply)
		ret = InternalErr
		return
	}

	for node, info := range NodeInfoMap {
		if info == nil || info.PubRPC == nil {
			glog.Errorf("abnormal node:\"%s\", interrupt pushing public message to node:\"%s\"", node)
			continue
		}

		// RPC call publish interface
		args := &myrpc.ChannelPushPublicArgs{MsgID: mid, Msg: string(body)}
		if err := info.PubRPC.Call("ChannelRPC.PushPublic", args, &ret); err != nil {
			glog.Errorf("RPC.Call(\"ChannelRPC.PushPublic\") server:\"%v\" error(%v)", info.Addr, err)
			ret = InternalErr
			return
		}
	}

	return
}

// AdminNodeAdd handle for add a node
func AdminNodeAdd(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final response operation
	defer func() {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		data, _ := json.Marshal(result)

		io.WriteString(rw, string(data))

		glog.Infof("request:node_add, request_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll() request_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		ret = ParamErr
		return
	}

	node := values.Get("node")
	if node == "node" {
		ret = ParamErr
		return
	}

	ret = OK
	return
}

// AdminNodeDel handle for del a node
func AdminNodeDel(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final response operation
	defer func() {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		data, _ := json.Marshal(result)

		io.WriteString(rw, string(data))

		glog.Infof("request:node_del, request_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll() request_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		ret = ParamErr
		return
	}

	node := values.Get("node")
	if node == "" {
		ret = ParamErr
		return
	}

	ret = OK
	return
}

// AdminCleanCache handle for clean the offline message of specified key
func AdminMsgClean(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final response operation
	var bodyStr string
	defer func(body *string) {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		data, _ := json.Marshal(result)

		io.WriteString(rw, string(data))

		glog.Infof("request:clean_cache, request_url:\"%s\", request_body:\"%s\", ret:\"%d\"", r.URL.String(), *body, ret)
	}(&bodyStr)

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Errorf("ioutil.ReadAll() request_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}
	bodyStr = string(body)

	values, err := url.ParseQuery(string(body))
	if err != nil {
		ret = ParamErr
		return
	}

	key := values.Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	// RPC call clean key interface
	reply, err := MessageRPCCleanKey(key)
	if err != nil {
		glog.Errorf("RPC.Call(\"ChannelRPC.CleanKey\") key:\"%s\" error(%v)", key, err)
		return
	}

	if reply != OK {
		ret = reply
		return
	}

	// Match a push-server with the value computed through ketama algorithm
	svrInfo := FindNode(key)
	if svrInfo == nil || svrInfo.PubRPC == nil {
		glog.Errorf("no node for key: \"%s\"", key)
		ret = NoNodeErr
		return
	}

	// RPC call ChannelRPC.Close interface
	if err := svrInfo.PubRPC.Call("ChannelRPC.Close", key, &ret); err != nil {
		glog.Errorf("RPC.Call(\"ChannelRPC.Close\") server:\"%v\" key:\"%s\" error(%v)", svrInfo.Addr, key, err)
		ret = InternalErr
		return
	}
}
