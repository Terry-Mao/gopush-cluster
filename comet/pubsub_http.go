package main

import (
	"code.google.com/p/go.net/websocket"
	"net"
	"net/http"
	"strconv"
	"time"
)

var (
	// websocket heartbeat
	websocketHeartbeatReply = []byte(Heartbeat)
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

func StartHttp() error {
	// set sub handler
	http.Handle("/sub", websocket.Handler(SubscribeHandle))
	if Conf.Debug == 1 {
		//http.HandleFunc("/client", Client)
	}

	if Conf.TCPKeepAlive == 1 {
		server := &http.Server{}
		l, err := net.Listen("tcp", Conf.Addr)
		if err != nil {
			Log.Error("net.Listen(\"tcp\", \"%s\") failed (%s)", Conf.Addr, err.Error())
			return err
		}

		Log.Info("start listen addr:%s", Conf.Addr)
		return server.Serve(&KeepAliveListener{Listener: l})
	} else {
		Log.Info("start listen addr:%s", Conf.Addr)
		if err := http.ListenAndServe(Conf.Addr, nil); err != nil {
			Log.Error("http.ListenAdServe(\"%s\") failed (%s)", Conf.Addr, err.Error())
			return err
		}
	}

	// nerve here
	return nil
}

// Subscriber Handle is the websocket handle for sub request.
func SubscribeHandle(ws *websocket.Conn) {
	params := ws.Request().URL.Query()
	// get subscriber key
	key := params.Get("key")
	if key == "" {
		Log.Warn("client:%s key param error", ws.Request().RemoteAddr)
		return
	}

	// get heartbeat second
	heartbeat := Conf.HeartbeatSec
	heartbeatStr := params.Get("heartbeat")
	if heartbeatStr != "" {
		i, err := strconv.Atoi(heartbeatStr)
		if err != nil {
			Log.Error("user_key:\"%s\" heartbeat argument error (%s)", key, err.Error())
			return
		}

		heartbeat = i
	}

	heartbeat *= 2
	if heartbeat <= 0 {
		Log.Error("user_key \"%s\" heartbeat argument error, less than 0", key)
		return
	}

	token := params.Get("token")
	Log.Info("client:%s subscribe to key = %s, heartbeat = %d, token = %s", ws.Request().RemoteAddr, key, heartbeat, token)
	// fetch subscriber from the channel
	c, err := UserChannel.Get(key)
	if err != nil {
		Log.Error("user_key:\"%s\" can't get a channel (%s)", key, err.Error())
		return
	}

	// auth token
	if Conf.Auth == 1 {
		if err = c.AuthToken(token, key); err != nil {
			Log.Error("user_key:\"%s\" auth token failed (%s)", key, err.Error())
			return
		}
	}

	// add a conn to the channel
	if err = c.AddConn(ws, key); err != nil {
		Log.Error("user_key:\"%s\" add conn failed (%s)", key, err.Error())
		return
	}

	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err = ws.Write(websocketHeartbeatReply); err != nil {
		Log.Error("user_key:\"%s\" write first heartbeat to client failed (%s)", key, err.Error())
		if err := c.RemoveConn(ws, key); err != nil {
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
			if _, err = ws.Write(websocketHeartbeatReply); err != nil {
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
	if err := c.RemoveConn(ws, key); err != nil {
		Log.Error("user_key:\"%s\" remove conn failed (%s)", key, err.Error())
	}

	return
}
