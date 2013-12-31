package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ServerGetData struct {
	Server string `json:"server"`
}

func ServerGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

	if r.Method != "GET" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	defer func() {
		result["ret"] = ret
		result["msg"] = GetErrMsg(ret)
		date, _ := json.Marshal(result)

		Log.Info("request:Get server, ret:%d", ret)

		io.WriteString(rw, string(date))
	}()

	key := r.URL.Query().Get("key")
	if key == "" {
		ret = ParamErr
		return
	}

	server := GetFirstServer(NodesHash.Node(key))
	if server == "" {
		ret = NoNodeErr
		return
	}

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

func MsgGet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret    = InternalErr
		result = make(map[string]interface{})
	)

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

	for i := 0; i < numMsg; i++ {
		if err := json.Unmarshal([]byte(msgs[i]), &msg); err != nil {
			Log.Error("internal message:%s error (%v)", msgs[i], err)
			ret = InternalErr
			return
		}

		// if message expired then ignore
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

func MsgSet(rw http.ResponseWriter, r *http.Request) {
	var (
		ret = InternalErr
	)

	if r.Method != "POST" {
		http.Error(rw, "Method Not Allowed", 405)
	}

	defer func() {
		rw.Header().Add("ret", strconv.Itoa(ret))
		rw.Header().Add("msg", GetErrMsg(ret))

		Log.Info("request:Set message, ret:%d", ret)

		io.WriteString(rw, "")
	}()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ret = InternalErr
		return
	}

	Log.Debug("%s", string(body))
	val, err := url.ParseQuery(string(body))
	if err != nil {
		ret = ParamErr
		return
	}

	key := val.Get("key")
	msg := val.Get("msg")
	if key == "" || msg == "" {
		ret = ParamErr
		return
	}

	expireI, err := strconv.ParseInt(val.Get("expire"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	midI, err := strconv.ParseInt(val.Get("mid"), 10, 64)
	if err != nil {
		ret = ParamErr
		return
	}

	recordMsg := Message{Msg: msg, Expire: expireI, MsgID: midI}
	message, _ := json.Marshal(recordMsg)
	if err := SaveMessage(key, string(message), midI); err != nil {
		Log.Error("save message error (%v)", err)
		ret = InternalErr
		return
	}

	ret = OK
	return
}
