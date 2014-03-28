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
		ret      = InternalErr
		result   = make(map[string]interface{})
		callback = ""
	)

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		data, _ := json.Marshal(result)

		Log.Info("request:Get_server, request_url:\"%s\", response_json:\"%s\", ip:\"%s\", ret:\"%d\"", r.URL.String(), string(data), r.RemoteAddr, ret)

		dataStr := ""
		if callback == "" {
			// Normal json
			dataStr = string(data)
		} else {
			// Jsonp
			dataStr = fmt.Sprintf("%s(%s)", callback, string(data))
		}
		io.WriteString(rw, dataStr)
	}()

	// Get params
	param := r.URL.Query()
	callback = param.Get("callback")
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
	svrInfo := FindNode(key)
	if svrInfo == nil {
		ret = NoNodeErr
		return
	}

	// Fill the server infomation into response json
	data := &ServerGetData{}
	addr := svrInfo.Addr[protoI]
	if addr == nil || len(addr) == 0 {
		ret = UnknownProtocol
		return
	}
	// TODO select the best ip
	data.Server = addr[0]
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
		ret      = InternalErr
		result   = make(map[string]interface{})
		callback = ""
	)

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		data, _ := json.Marshal(result)

		Log.Info("request:Get_messages, request_url:\"%s\", response_json:\"%s\"), ip:\"%s\", ret:\"%d\"", r.URL.String(), string(data), r.RemoteAddr, ret)

		dataStr := ""
		if callback == "" {
			// Normal json
			dataStr = string(data)
		} else {
			// Jsonp
			dataStr = fmt.Sprintf("%s(%s)", callback, string(data))
		}
		io.WriteString(rw, dataStr)
	}()

	// Get params
	val := r.URL.Query()
	callback = val.Get("callback")
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

	// TODO: If one of midI or pMidI is 0, then respond nil message,
	// for avoid client`s bug that always use the default 0 of integer variable,so that receive the repetitive messages.
	// After the client installed, the first it needs to request url /time/get and get the initial message id
	//if midI == 0 || pMidI == 0 {
	//	ret = OK
	//	return
	//}

	// RPC get offline messages
	reply, err := MessageRPCGet(key, midI, pMidI)
	if err != nil {
		Log.Error("RPC.Call(\"MessageRPC.Get\")  Key:\"%s\", MsgID:\"%d\" error(%v)", key, midI, err)
		ret = InternalErr
		return
	}

	if reply.Ret != OK {
		Log.Error("RPC.Call(\"MessageRPC.Get\")  Key:\"%s\", MsgID:\"%d\" errorCode(\"%d\")", key, midI, reply.Ret)
		ret = reply.Ret
		return
	}

	if len(reply.Msgs) == 0 && len(reply.PubMsgs) == 0 {
		ret = OK
		return
	}

	data := &MsgGetData{}
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

// Data struct as response of handle TimeGetData
type TimeGetData struct {
	TimeID int64 `json:"timeid"` // as message id
}

// Get server time
func TimeGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret      = InternalErr
		result   = make(map[string]interface{})
		callback = ""
	)

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
		return
	}

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		data, _ := json.Marshal(result)

		Log.Info("request:Get_server_time, request_url:\"%s\", response_json:\"%s\", ip:\"%s\", ret:\"%d\"", r.URL.String(), string(data), r.RemoteAddr, ret)

		dataStr := ""
		if callback == "" {
			// Normal json
			dataStr = string(data)
		} else {
			// Jsonp
			dataStr = fmt.Sprintf("%s(%s)", callback, string(data))
		}
		io.WriteString(rw, dataStr)
	}()

	val := r.URL.Query()
	callback = val.Get("callback")

	//result["data"] = &TimeGetData{TimeID: PubMID.ID()}
	ret = OK
}
