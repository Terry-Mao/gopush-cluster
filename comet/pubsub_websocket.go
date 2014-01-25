package main

import (
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type KeepAliveListener struct {
	net.Listener
}

func (l *KeepAliveListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		Log.Error("Listener.Accept() failed (%s)", err.Error())
		return
	}
	// set keepalive
	if tc, ok := c.(*net.TCPConn); !ok {
		Log.Crit("net.TCPConn assection type failed")
		panic("Assection type failed c.(net.TCPConn)")
	} else {
		err = tc.SetKeepAlive(true)
		if err != nil {
			Log.Error("tc.SetKeepAlive(true) failed (%s)", err.Error())
			return
		}
	}
	return
}

// StartHttp start http listen.
func StartHttp() {
	for _, bind := range strings.Split(Conf.WebsocketBind, ",") {
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
			Log.Error("net.Listen(\"tcp\", \"%s\") failed (%s)", bind, err.Error())
			os.Exit(-1)
		}
		if err := server.Serve(&KeepAliveListener{Listener: l}); err != nil {
			Log.Error("server.Serve(\"%s\") failed (%s)", bind, err.Error())
			os.Exit(-1)
		}
	} else {
		if err := http.ListenAndServe(bind, httpServeMux); err != nil {
			Log.Error("http.ListenAdServe(\"%s\") failed (%s)", bind, err.Error())
			os.Exit(-1)
		}
	}
}

// Subscriber Handle is the websocket handle for sub request.
func SubscribeHandle(ws *websocket.Conn) {
	params := ws.Request().URL.Query()
	// get subscriber key
	key := params.Get("key")
	if key == "" {
		ws.Write(ParamReply)
		Log.Warn("client:%s key param error", ws.Request().RemoteAddr)
		return
	}
	// get heartbeat second
	heartbeatStr := params.Get("heartbeat")
	i, err := strconv.Atoi(heartbeatStr)
	if err != nil {
		ws.Write(ParamReply)
		Log.Error("user_key:\"%s\" heartbeat argument error (%s)", key, err.Error())
		return
	}
	heartbeat := i * 2
	if heartbeat <= minHearbeatSec {
		ws.Write(ParamReply)
		Log.Error("user_key \"%s\" heartbeat argument error, less than 0", key)
		return
	}
	token := params.Get("token")
	Log.Info("client:%s subscribe to key = %s, heartbeat = %d, token = %s", ws.Request().RemoteAddr, key, heartbeat, token)
	// fetch subscriber from the channel
	c, err := UserChannel.Get(key)
	if err != nil {
		ws.Write(ChannelReply)
		Log.Error("user_key:\"%s\" can't get a channel (%s)", key, err.Error())
		return
	}
	// auth token
	if ok := c.AuthToken(key, token); !ok {
		ws.Write(AuthReply)
		Log.Error("user_key:\"%s\" auth token \"%s\" failed", key, token)
		return
	}
	// add a conn to the channel
	connElem, err := c.AddConn(ws, key)
	if err != nil {
		Log.Error("user_key:\"%s\" add conn failed (%s)", key, err.Error())
		return
	}
	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err = ws.Write(HeartbeatReply); err != nil {
		Log.Error("user_key:\"%s\" write first heartbeat to client failed (%s)", key, err.Error())
		if err := c.RemoveConn(connElem, key); err != nil {
			Log.Error("user_key:\"%s\" remove conn failed (%s)", key, err.Error())
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
				Log.Error("user_key:\"%s\" websocket.SetReadDeadline() failed (%s)", key, err.Error())
				break
			}
			begin = end
		}
		if err = websocket.Message.Receive(ws, &reply); err != nil {
			Log.Error("user_key:\"%s\" websocket.Message.Receive() failed (%s)", key, err.Error())
			break
		}
		if reply == Heartbeat {
			if _, err = ws.Write(HeartbeatReply); err != nil {
				Log.Error("user_key:\"%s\" write heartbeat to client failed (%s)", key, err.Error())
				break
			}
			Log.Debug("user_key:\"%s\" receive heartbeat", key)
		} else {
			Log.Warn("user_key:\"%s\" unknown heartbeat protocol", key)
			break
		}
		end = time.Now().UnixNano()
	}
	// remove exists conn
	if err := c.RemoveConn(connElem, key); err != nil {
		Log.Error("user_key:\"%s\" remove conn failed (%s)", key, err.Error())
	}
	return
}
