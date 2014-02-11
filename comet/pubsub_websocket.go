package main

import (
	"code.google.com/p/go.net/websocket"
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
		Log.Error("Listener.Accept() error(%v)", err)
		return
	}
	// set keepalive
	if tc, ok := c.(*net.TCPConn); !ok {
		Log.Crit("net.TCPConn assection type failed")
		panic("Assection type failed c.(net.TCPConn)")
	} else {
		err = tc.SetKeepAlive(true)
		if err != nil {
			Log.Error("tc.SetKeepAlive(true) error(%v)", err)
			return
		}
	}
	return
}

// StartHttp start http listen.
func StartHttp() {
	for _, bind := range Conf.WebsocketBind {
		Log.Info("start http listen addr:\"%s\"", bind)
		go httpListen(bind)
	}
}

func httpListen(bind string) {
	httpServeMux := http.NewServeMux()
	httpServeMux.Handle("/sub", websocket.Handler(SubscribeHandle))
	if Conf.TCPKeepalive {
		server := &http.Server{Handler: httpServeMux}
		l, err := net.Listen("tcp", bind)
		if err != nil {
			Log.Error("net.Listen(\"tcp\", \"%s\") error(%v)", bind, err)
			panic(err)
		}
		if err := server.Serve(&KeepAliveListener{Listener: l}); err != nil {
			Log.Error("server.Serve(\"%s\") error(%v)", bind, err)
			panic(err)
		}
	} else {
		if err := http.ListenAndServe(bind, httpServeMux); err != nil {
			Log.Error("http.ListenAdServe(\"%s\") error(%v)", bind, err)
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
		Log.Warn("<%s> key param error", addr)
		return
	}
	// get heartbeat second
	heartbeatStr := params.Get("heartbeat")
	i, err := strconv.Atoi(heartbeatStr)
	if err != nil {
		ws.Write(ParamReply)
		Log.Error("<%s> user_key:\"%s\" heartbeat argument error(%s)", addr, key, err)
		return
	}
	heartbeat := i * 2
	if heartbeat <= minHearbeatSec {
		ws.Write(ParamReply)
		Log.Error("<%s> user_key \"%s\" heartbeat argument error, less than 0", addr, key)
		return
	}
	token := params.Get("token")
	Log.Info("<%s> subscribe to key = %s, heartbeat = %d, token = %s", addr, key, heartbeat, token)
	// fetch subscriber from the channel
	c, err := UserChannel.Get(key)
	if err != nil {
		ws.Write(ChannelReply)
		Log.Error("<%s> user_key:\"%s\" can't get a channel error(%s)", addr, key, err)
		return
	}
	// auth token
	if ok := c.AuthToken(key, token); !ok {
		ws.Write(AuthReply)
		Log.Error("<%s> user_key:\"%s\" auth token \"%s\" failed", addr, key, token)
		return
	}
	// add a conn to the channel
	connElem, err := c.AddConn(key, ws)
	if err != nil {
		Log.Error("<%s> user_key:\"%s\" add conn error(%s)", addr, key, err)
		return
	}
	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err = ws.Write(HeartbeatReply); err != nil {
		Log.Error("<%s> user_key:\"%s\" write first heartbeat to client error(%s)", addr, key, err)
		if err := c.RemoveConn(key, connElem); err != nil {
			Log.Error("<%s> user_key:\"%s\" remove conn error(%v)", addr, key, err)
		}
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
				Log.Error("<%s> user_key:\"%s\" websocket.SetReadDeadline() error(%s)", addr, key, err)
				break
			}
			begin = end
		}
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			Log.Error("<%s> user_key:\"%s\" websocket.Message.Receive() error(%s)", addr, key, err)
			break
		}
		if reply == Heartbeat {
			if _, err = ws.Write(HeartbeatReply); err != nil {
				Log.Error("<%s> user_key:\"%s\" write heartbeat to client error(%s)", addr, key, err)
				break
			}
			Log.Debug("<%s> user_key:\"%s\" receive heartbeat", addr, key)
		} else {
			Log.Warn("<%s> user_key:\"%s\" unknown heartbeat protocol", addr, key)
			break
		}
		end = time.Now().UnixNano()
	}
	// remove exists conn
	if err := c.RemoveConn(key, connElem); err != nil {
		Log.Error("<%s> user_key:\"%s\" remove conn error(%s)", addr, key, err)
	}
	return
}
