package main

import (
	"encoding/json"
	myrpc "github.com/Terry-Mao/gopush-cluster/rpc"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
)

// ServerPush handle for server push
func ServerPush(rw http.ResponseWriter, r *http.Request) {
	var (
		bRW    = false
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	// Final response operation
	defer func() {
		if bRW == false {
			result["msg"] = GetErrMsg(ret)
			result["ret"] = ret
			date, _ := json.Marshal(result)

			io.WriteString(rw, string(date))

			Log.Info("request:push, quest_url:%s, ret:%d", r.URL.String(), ret)
		}
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
	svrInfo := GetFirstServer(CometHash.Node(key))
	if svrInfo == nil {
		ret = NoNodeErr
		return
	}

	// RPC call publish interface
	args := &myrpc.ChannelPublishArgs{MsgID: mid, Msg: string(body), Expire: expire, Key: key}
	if err := svrInfo.PubRPC.Call("ChannelRPC.Publish", args, &ret); err != nil {
		Log.Error("RPC.Call(\"ChannelRPC.Publish\") error server:%s (%v)", svrInfo.Data, err)
		ret = InternalErr
		return
	}

	return
}

// Data struct as response of handle ServerGet
type ServerGetData struct {
	Server string `json:"server"`
}

// ServerGet handle for server get
func ServerGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		date, _ := json.Marshal(result)

		Log.Info("request:Get server, quest_url:%s, ret:%d", r.URL.String(), ret)

		io.WriteString(rw, string(date))
	}()

	// Get params
	key := r.URL.Query().Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	// Match a push-server with the value computed through ketama algorithm
	svrInfo := GetFirstServer(CometHash.Node(key))
	if svrInfo == nil {
		ret = NoNodeErr
		return
	}

	// Fill the server infomation into response json
	data := &ServerGetData{}
	data.Server = svrInfo.Data

	result["data"] = data
	ret = OK
	return
}

// Data struct as response of handle ServerGet
type MsgGetData struct {
	Msgs []string `json:"msgs"`
}

// MsgGet handle for msg get
func MsgGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		date, _ := json.Marshal(result)

		Log.Info("request:Get messages, quest_url:%s, ret:%d", r.URL.String(), ret)

		io.WriteString(rw, string(date))
	}()

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	// Get params
	val := r.URL.Query()
	key := val.Get("key")
	mid := val.Get("mid")
	if key == "" || mid == "" {
		ret = ParamErr
		return
	}

	midI, err := strconv.ParseInt(mid, 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	var reply myrpc.MessageGetResp
	args := &myrpc.MessageGetArgs{MsgID: midI, Key: key}
	if err := MsgSvrClient.Call("MessageRPC.Get", args, &reply); err != nil {
		Log.Error("RPC.Call(\"MessageRPC.Get\") error MsgID:%d, Key:%s (%v)", midI, key, err)
		ret = InternalErr
		return
	}

	if len(reply.Msgs) > 0 {
		result["data"] = reply.Msgs
	}

	return
}
