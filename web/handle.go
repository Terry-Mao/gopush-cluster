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

		Log.Info("request:Get server, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)

		io.WriteString(rw, string(date))
	}()

	// Get params
	param := r.URL.Query()
	key := param.Get("key")

	if key == "" {
		ret = ParamErr
		return
	}

	protoI, err := strconv.Atoi(param.Get("proto"))
	if err != nil {
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
	addr := svrInfo.SubAddr[protoI]
	if addr == "" {
		ret = UnknownProtocol
		return
	}
	data.Server = addr

	result["data"] = data
	ret = OK
	return
}

// Data struct as response of handle ServerGet
type MsgGetData struct {
	Msgs  []string `json:"msgs"`
	PMsgs []string `json:"pmsgs"`
}

// MsgGet handle for msg get
func MsgGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
		data   = &MsgGetData{}
	)

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		date, _ := json.Marshal(result)

		Log.Info("request:Get messages, quest_url:\"%s\", ret:\"%d\"", r.URL.String(), ret)

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
	pMid := val.Get("pmid")
	if key == "" || mid == "" || pMid == "" {
		ret = ParamErr
		return
	}

	midI, err := strconv.ParseInt(mid, 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	pMidI, err := strconv.ParseInt(pMid, 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	// RPC get offline messages
	reply, err := MessageRPCGet(key, midI, pMidI)
	if err != nil {
		Log.Error("RPC.Call(\"MessageRPC.Get\")  Key:\"%s\", MsgID:\"%d\" error(%v)", key, midI, err)
		ret = InternalErr
		return
	}

	if reply.Ret != OK {
		Log.Error("RPC.Call(\"MessageRPC.Get\")  Key:\"%s\", MsgID:\"%d\" errorCode(%d)", key, midI, reply.Ret)
		ret = reply.Ret
		return
	}

	if len(reply.Msgs) == 0 && len(reply.PubMsgs) == 0 {
		ret = OK
		return
	}

	if len(reply.Msgs) > 0 {
		data.Msgs = reply.Msgs
	}

	if len(reply.PubMsgs) > 0 {
		data.PMsgs = reply.PubMsgs
	}

	result["data"] = data
	ret = reply.Ret
	return
}
