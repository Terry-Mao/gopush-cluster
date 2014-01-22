package main

import (
	"encoding/json"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

// AdminPush handle for push message
func AdminPush(rw http.ResponseWriter, r *http.Request) {
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
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push, quest_url:%s, ret:%d", r.URL.String(), ret)
	}()

	// Get params
	param := r.URL.Query()
	key := param.Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	mid, err := strconv.ParseInt(param.Get("mid"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	expire, err := strconv.ParseInt(param.Get("expire"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() error quest_url:%s (%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}

	// Match a push-server with the value computed through ketama algorithm
	svrInfo := GetNode(CometHash.Node(key))
	if svrInfo == nil {
		ret = NoNodeErr
		return
	}

	// RPC call publish interface
	args := &myrpc.ChannelPublishArgs{MsgID: mid, Msg: string(body), Expire: expire, Key: key}
	if err := svrInfo.PubRPC.Call("ChannelRPC.Publish", args, &ret); err != nil {
		Log.Error("RPC.Call(\"ChannelRPC.Publish\") error server:%s (%v)", svrInfo.SubAddr, err)
		ret = InternalErr
		return
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
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push, quest_url:%s, ret:%d", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() error quest_url:%s (%v)", r.URL.String(), err)
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

	// Add a node
	if err := AddNode(node); err != nil {
		if err == ErrNodeExist {
			ret = NodeExist
		} else {
			ret = InternalErr
		}

		Log.Error("add node error (%v)", err)
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
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push, quest_url:%s, ret:%d", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() error quest_url:%s (%v)", r.URL.String(), err)
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

	// Add a watch for node
	if err := DelNode(node); err != nil {
		Log.Error("del node error (%v)", err)
		ret = InternalErr
		return
	}

	ret = OK
	return
}
