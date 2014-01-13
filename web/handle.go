package main

import (
	"encoding/json"

	"io"
	"net/http"
	"strconv"
)

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
		return
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
	svrInfo := GetNode(CometHash.Node(key))
	if svrInfo == nil {
		ret = NoNodeErr
		return
	}

	// Fill the server infomation into response json
	data := &ServerGetData{}
	data.Server = svrInfo.SubAddr

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
		return
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

	// RPC get offline messages
	reply, err := MessageRPCGet(midI, key)
	if err != nil {
		Log.Error("RPC.Call(\"MessageRPC.Get\") error MsgID:%d, Key:%s (%v)", midI, key, err)
		ret = InternalErr
		return
	}

	if len(reply.Msgs) > 0 {
		result["data"] = reply.Msgs
	}

	ret = reply.Ret
	return
}
