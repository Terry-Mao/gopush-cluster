package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
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
	}

	// Final ResponseWriter operation
	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		date, _ := json.Marshal(result)

		Log.Info("request:Get server, ret:%d", ret)

		io.WriteString(rw, string(date))
	}()

	// Get params
	Log.Debug("request_url:%s", r.URL.String())
	key := r.URL.Query().Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	// Match a push-server with the value computed through ketama algorithm
	server := GetFirstServer(CometHash.Node(key))
	if server == "" {
		ret = NoNodeErr
		return
	}

	// Fill the server infomation into response json
	data := &ServerGetData{}
	data.Server = server

	result["data"] = data
	ret = OK
	return
}

// The Message struct
type Message struct {
	// Message
	Msg string `json:"msg"`
	// Message expired unixnano
	Expire int64 `json:"expire"`
	// Message id
	MsgID int64 `json:"mid"`
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

		Log.Info("request:Get messages, ret:%d", ret)

		io.WriteString(rw, string(date))
	}()

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	// Get params
	Log.Debug("request_url:%s", r.URL.String())
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

	// Get all of offline messages which larger than midI
	msgs, err := GetMessages(key, midI)
	if err != nil {
		Log.Error("get messages error (%v)", err)
		ret = InternalErr
		return
	}

	numMsg := len(msgs)
	if len(msgs) == 0 {
		ret = OK
		return
	}

	var (
		data []string
		msg  = &Message{}
	)

	// Checkout expired offline messages
	for i := 0; i < numMsg; i++ {
		if err := json.Unmarshal([]byte(msgs[i]), &msg); err != nil {
			Log.Error("internal message:%s error (%v)", msgs[i], err)
			ret = InternalErr
			return
		}

		if time.Now().UnixNano() > msg.Expire {
			continue
		}

		data = append(data, msgs[i])
	}

	if len(data) == 0 {
		ret = OK
		return
	}

	result["data"] = data
	ret = OK
	return
}
