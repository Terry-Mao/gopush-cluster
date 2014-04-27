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
	"code.google.com/p/go.net/websocket"
	"github.com/golang/glog"
	"net"
	"net/http"
	"strconv"
	"time"
)

type KeepAliveListener struct {
	net.Listener
}

func (l *KeepAliveListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		glog.Errorf("Listener.Accept() error(%v)", err)
		return
	}
	// set keepalive
	if tc, ok := c.(*net.TCPConn); !ok {
		glog.Error("net.TCPConn assection type failed")
		panic("Assection type failed c.(net.TCPConn)")
	} else {
		err = tc.SetKeepAlive(true)
		if err != nil {
			glog.Errorf("tc.SetKeepAlive(true) error(%v)", err)
			return
		}
	}
	return
}

// StartHttp start http listen.
func StartHttp() {
	for _, bind := range Conf.WebsocketBind {
		glog.Infof("start websocket listen addr:\"%s\"", bind)
		go websocketListen(bind)
	}
}

func websocketListen(bind string) {
	httpServeMux := http.NewServeMux()
	httpServeMux.Handle("/sub", websocket.Handler(SubscribeHandle))
	if Conf.TCPKeepalive {
		server := &http.Server{Handler: httpServeMux}
		l, err := net.Listen("tcp", bind)
		if err != nil {
			glog.Errorf("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
			panic(err)
		}
		if err := server.Serve(&KeepAliveListener{Listener: l}); err != nil {
			glog.Errorf("server.Serve(\"%s\") error(%v)", bind, err)
			panic(err)
		}
	} else {
		if err := http.ListenAndServe(bind, httpServeMux); err != nil {
			glog.Errorf("http.ListenAdServe(\"%s\") error(%v)", bind, err)
			panic(err)
		}
	}
}

// Subscriber Handle is the websocket handle for sub request.
func SubscribeHandle(ws *websocket.Conn) {
	addr := ws.Request().RemoteAddr
	params := ws.Request().URL.Query()
	// get subscriber key
	key := params.Get("key")
	if key == "" {
		ws.Write(ParamReply)
		glog.Warningf("<%s> key param error", addr)
		return
	}
	// get heartbeat second
	heartbeatStr := params.Get("heartbeat")
	i, err := strconv.Atoi(heartbeatStr)
	if err != nil {
		ws.Write(ParamReply)
		glog.Errorf("<%s> user_key:\"%s\" heartbeat argument error(%s)", addr, key, err)
		return
	}
	if i < minHearbeatSec {
		ws.Write(ParamReply)
		glog.Warningf("<%s> user_key:\"%s\" heartbeat argument error, less than %d", addr, key, minHearbeatSec)
		return
	}
	heartbeat := i + delayHeartbeatSec
	token := params.Get("token")
	version := params.Get("ver")
	glog.Infof("<%s> subscribe to key = %s, heartbeat = %d, token = %s, version = %s", addr, key, heartbeat, token, version)
	// fetch subscriber from the channel
	c, err := UserChannel.Get(key, true)
	if err != nil {
		ws.Write(ChannelReply)
		glog.Errorf("<%s> user_key:\"%s\" can't get a channel error(%s)", addr, key, err)
		return
	}
	// auth token
	if ok := c.AuthToken(key, token); !ok {
		ws.Write(AuthReply)
		glog.Errorf("<%s> user_key:\"%s\" auth token \"%s\" failed", addr, key, token)
		return
	}
	// add a conn to the channel
	connElem, err := c.AddConn(key, &Connection{Conn: ws, Proto: WebsocketProto, Version: version})
	if err != nil {
		glog.Errorf("<%s> user_key:\"%s\" add conn error(%s)", addr, key, err)
		return
	}
	// blocking wait client heartbeat
	reply := ""
	begin := time.Now().UnixNano()
	end := begin + Second
	for {
		// more then 1 sec, reset the timer
		if end-begin >= Second {
			if err = ws.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat))); err != nil {
				glog.Errorf("<%s> user_key:\"%s\" websocket.SetReadDeadline() error(%s)", addr, key, err)
				break
			}
			begin = end
		}
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			glog.Errorf("<%s> user_key:\"%s\" websocket.Message.Receive() error(%s)", addr, key, err)
			break
		}
		if reply == Heartbeat {
			if _, err = ws.Write(HeartbeatReply); err != nil {
				glog.Errorf("<%s> user_key:\"%s\" write heartbeat to client error(%s)", addr, key, err)
				break
			}
			glog.V(1).Infof("<%s> user_key:\"%s\" receive heartbeat", addr, key)
		} else {
			glog.Warningf("<%s> user_key:\"%s\" unknown heartbeat protocol", addr, key)
			break
		}
		end = time.Now().UnixNano()
	}
	// remove exists conn
	if err := c.RemoveConn(key, connElem); err != nil {
		glog.Errorf("<%s> user_key:\"%s\" remove conn error(%s)", addr, key, err)
	}
	return
}
