package main

import (
	"encoding/json"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
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
	defer func() {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push_private, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	param := r.URL.Query()
	key := param.Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	groupID, err := strconv.Atoi(param.Get("gid"))
	if err != nil {
		ret = ParamErr
		return
	}

	expire, err := strconv.ParseInt(param.Get("expire"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	// Get message
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() quest_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}

	// Match a push-server with the value computed through ketama algorithm
	svrInfo := GetNode(CometHash.Node(key))
	if svrInfo == nil {
		Log.Debug("no node:\"%s\"", CometHash.Node(key))
		ret = NoNodeErr
		return
	}

	// RPC call publish interface
	args := &myrpc.ChannelPushPrivateArgs{GroupID: groupID, Msg: string(body), Expire: expire, Key: key}
	if err := svrInfo.PubRPC.Call("ChannelRPC.PushPrivate", args, &ret); err != nil {
		Log.Error("RPC.Call(\"ChannelRPC.PushPrivate\") server:\"%v\" error(%v)", svrInfo.SubAddr, err)
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
	defer func() {
		result["msg"] = GetErrMsg(ret)
		result["ret"] = ret
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push_public, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	param := r.URL.Query()
	expire, err := strconv.ParseInt(param.Get("expire"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() quest_url:\"%s\" error(%v)", r.URL.String(), err)
		ret = InternalErr
		return
	}

	// Lock here, make sure that get the unique mid
	lockYes, pathCreated, err := PubMID.Lock()
	if pathCreated != "" {
		defer PubMID.LockRelease(pathCreated)
	}
	if err != nil || lockYes == false {
		Log.Error("PubMIDLock error(%v)", err)
		ret = InternalErr
		return
	}
	mid := PubMID.ID()

	// Save public message
	expire = time.Now().Add(time.Duration(expire) * time.Second).UnixNano()
	reply, err := MessageRPCSavePub(string(body), mid, expire)
	// Message save failed
	if reply != OK {
		Log.Error("RPC.Call(\"MessageRPC.SavePub\") error (ret:\"%d\")", reply)
		ret = InternalErr
		return
	}

	for node, info := range NodeInfoMap {
		if info == nil {
			Log.Error("abnormal node:\"%s\", do not push public message to node:\"%s\"", node)
			continue
		}

		// RPC call publish interface
		args := &myrpc.ChannelPushPublicArgs{MsgID: mid, Msg: string(body)}
		if err := info.PubRPC.Call("ChannelRPC.PushPublic", args, &ret); err != nil {
			Log.Error("RPC.Call(\"ChannelRPC.PushPublic\") server:\"%v\" error(%v)", info.SubAddr, err)
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
		date, _ := json.Marshal(result)

		io.WriteString(rw, string(date))

		Log.Info("request:push, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() quest_url:\"%s\" error(%v)", r.URL.String(), err)
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

		Log.Error("add node:\"%s\" error(%v)", node, err)
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

		Log.Info("request:push, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)
	}()

	// Get params
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Log.Error("ioutil.ReadAll() quest_url:\"%s\" error(%v)", r.URL.String(), err)
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
	Log.Debug("del node:\"%s\"", node)
	if err := DelNode(node); err != nil {
		Log.Error("del node error(%v)", err)
		ret = InternalErr
		return
	}

	ret = OK
	return
}
