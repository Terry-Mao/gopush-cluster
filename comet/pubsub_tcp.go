package main

import (
	"bufio"
	"errors"
	"io"
	"net"
	"strconv"
	"time"
)

const (
	fitstPacketTimedoutSec = 5
)

var (
	CmdFmtErr = errors.New("cmd format error")
)

// TCPReadBuf store buffer in chan
type TCPReadBuf struct {
	instance []chan *bufio.Reader
}

// NewTCPReadBuf get a mutiple instance chan stored the bufio.Reader obj
func NewTCPReadBuf() *TCPReadBuf {
	readBufInstance := make([]chan *bufio.Reader, 0, Conf.ReadBufInstance)
	Log.Debug("create %d read buffer instance", Conf.ReadBufInstance)
	for i := 0; i < Conf.ReadBufInstance; i++ {
		readBufInstance = append(readBufInstance, make(chan *bufio.Reader, Conf.ReadBufNumPerInst))
	}

	return &TCPReadBuf{instance: readBufInstance}
}

// Get return a bufio.Reader, if chan is empty then create a new one
func (b *TCPReadBuf) Get(conn io.Reader, idx int) *bufio.Reader {
	select {
	case rd := <-b.instance[idx]:
		rd.Reset(conn)
		return rd
	default:
		Log.Debug("tcp read buffer empty")
		return bufio.NewReaderSize(conn, Conf.ReadBufByte)
	}
}

// Pub return back a reader to the chan, if chan is full then discard it
func (b *TCPReadBuf) Put(buf *bufio.Reader, idx int) {
	buf.Reset(nil)
	select {
	case b.instance[idx] <- buf:
	default:
		Log.Debug("tcp read buffer full")
	}
}

// StartTCP start listen tcp
func StartTCP() error {
	addr, err := net.ResolveTCPAddr("tcp", Conf.Addr)
	if err != nil {
		Log.Error("net.ResolveTCPAddr(\"tcp\"), %s) failed (%s)", Conf.Addr, err.Error())
		return err
	}

	Log.Info("start listen addr:%s", addr)
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		Log.Error("net.ListenTCP(\"tcp4\", \"%s\") failed (%s)", Conf.Addr, err.Error())
		return err
	}

	// free the listener resource
	defer func() {
		if err := l.Close(); err != nil {
			Log.Error("listener.Close() failed (%s)", err.Error())
		}
	}()

	// init reader buffer instance
	rb := NewTCPReadBuf()
	// loop for accept conn
	round := 0
	for {
		Log.Debug("start accept")
		conn, err := l.AcceptTCP()
		if err != nil {
			Log.Error("listener.AcceptTCP() failed (%s)", err.Error())
			continue
		}

		if err = conn.SetKeepAlive(Conf.TCPKeepAlive == 1); err != nil {
			Log.Error("conn.SetKeepAlive() failed (%s)", err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetReadBuffer(Conf.ReadBufByte); err != nil {
			Log.Error("conn.SetReadBuffer(%d) failed (%s)", Conf.ReadBufByte, err.Error())
			conn.Close()
			continue
		}

		if err = conn.SetWriteBuffer(Conf.WriteBufByte); err != nil {
			Log.Error("conn.SetWriteBuffer(%d) failed (%s)", Conf.WriteBufByte, err.Error())
			conn.Close()
			continue
		}

		// first packet must sent by client in 5 seconds
		if err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(fitstPacketTimedoutSec))); err != nil {
			Log.Error("conn.SetReadDeadLine() failed (%s)", err.Error())
			conn.Close()
			continue
		}

		// one connection one routine
		go handleTCPConn(conn, round, rb)
		// split requets to diff buffer chan
		round++
		if round == Conf.ReadBufInstance {
			round = 0
		}

		Log.Debug("accept finished")
	}

	// nerve here
	Log.Crit("touch a impossible place")
	return nil
}

// hanleTCPConn handle a long live tcp connection
func handleTCPConn(conn net.Conn, round int, rb *TCPReadBuf) {
	Log.Debug("handleTcpConn routine start")
	// parse protocol reference: http://redis.io/topics/protocol (use redis protocol)
	rd := rb.Get(conn, round)
	if args, err := parseCmd(rd); err == nil {
		// return buffer bufio.Reader
		rb.Put(rd, round)
		switch args[0] {
		case "sub":
			SubscribeTCPHandle(conn, args[1:])
			break
		default:
			Log.Warn("tcp:unknown cmd \"%s\"", args[0])
			break
		}
	} else {
		// return buffer bufio.Reader
		rb.Put(rd, round)
		Log.Error("parseCmd() failed (%s)", err.Error())
	}

	// close the connection
	if err := conn.Close(); err != nil {
		Log.Error("conn.Close() failed (%s)", err.Error())
	}

	Log.Debug("handleTcpConn routine stop")
	return
}

// SubscribeTCPHandle handle the subscribers's connection
func SubscribeTCPHandle(conn net.Conn, args []string) {
	argLen := len(args)
	if argLen < 2 {
		Log.Error("subscriber missing argument")
		return
	}

	// key, mid, heartbeat
	key := args[0]
	if key == "" {
		Log.Warn("client:%s key param error", conn.RemoteAddr)
		return
	}

	midStr := args[1]
	mid, err := strconv.ParseInt(midStr, 10, 64)
	if err != nil {
		Log.Error("user_key:\"%s\" parse mid:\"%s\" error (%s)", key, midStr, err.Error())
		return
	}

	heartbeat := Conf.HeartbeatSec
	heartbeatStr := ""
	if argLen > 2 {
		heartbeatStr = args[2]
		i, err := strconv.Atoi(heartbeatStr)
		if err != nil {
			Log.Error("user_key:\"%s\" heartbeat:\"%s\" argument error (%s)", key, heartbeatStr, err.Error())
			return
		}

		heartbeat = i
	}

	heartbeat *= 2
	if heartbeat <= 0 {
		Log.Warn("user_key:\"%s\" heartbeat argument error, less than 0", key)
		return
	}

	Log.Info("client:\"%s\" subscribe to key = %s, mid = %d, heartbeat = %d", conn.RemoteAddr().String(), key, mid, heartbeat)
	// fetch subscriber from the channel
	c, err := UserChannel.Get(key)
	if err != nil {
		Log.Warn("user_key:\"%s\" can't get a channel (%s)", key, err.Error())
		return
	}

	// send first heartbeat to tell client service is ready for accept heartbeat
	if _, err := conn.Write(TCPHeartbeatReply); err != nil {
		Log.Error("user_key:\"%s\" write first heartbeat to client failed (%s)", key, err.Error())
		return
	}

	// send stored message, and use the last message id if sent any
	if err = c.SendOfflineMsg(conn, mid, key); err != nil {
		Log.Error("user_key:\"%s\" send offline message failed (%s)", key, err.Error())
		return
	}

	// add a conn to the channel
	if err = c.AddConn(conn, mid, key); err != nil {
		Log.Error("user_key:\"%s\" add conn failed (%s)", key, err.Error())
		return
	}

	// blocking wait client heartbeat
	reply := make([]byte, HeartbeatLen)
	begin := time.Now().UnixNano()
	end := begin + oneSecond
	for {
		// more then 1 sec, reset the timer
		if end-begin >= oneSecond {
			if err = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat))); err != nil {
				Log.Error("user_key:\"%s\" conn.SetReadDeadLine() failed (%s)", key, err.Error())
				break
			}

			begin = end
		}

		if _, err = conn.Read(reply); err != nil {
			if err != io.EOF {
				Log.Warn("user_key:\"%s\" conn.Read() failed, read heartbeat timedout (%s)", key, err.Error())
			} else {
				// client connection close
				Log.Warn("user_key:\"%s\" client connection close (%s)", key, err.Error())
			}

			break
		}

		if string(reply) == Heartbeat {
			if _, err = conn.Write(TCPHeartbeatReply); err != nil {
				Log.Error("user_key:\"%s\" conn.Write() failed, write heartbeat to client (%s)", key, err.Error())
				break
			}

			Log.Debug("user_key:\"%s\" receive heartbeat (%s)", key, reply)
		} else {
			Log.Warn("user_key:\"%s\" unknown heartbeat protocol (%s)", key, reply)
			break
		}

		end = time.Now().UnixNano()
	}

	// remove exists conn
	if err := c.RemoveConn(conn, mid, key); err != nil {
		Log.Error("user_key:\"%s\" remove conn failed (%s)", key, err.Error())
	}

	return
}

// parseCmd parse the tcp request command
func parseCmd(rd *bufio.Reader) ([]string, error) {
	// get argument number
	argNum, err := parseCmdSize(rd, '*')
	if err != nil {
		Log.Error("tcp:cmd format error when find '*' (%s)", err.Error())
		return nil, err
	}

	if argNum < 1 {
		Log.Error("tcp:cmd argument number length error")
		return nil, CmdFmtErr
	}

	args := make([]string, 0, argNum)
	for i := 0; i < argNum; i++ {
		// get argument length
		cmdLen, err := parseCmdSize(rd, '$')
		if err != nil {
			Log.Error("tcp:parseCmdSize(rd, '$') failed (%s)", err.Error())
			return nil, err
		}

		// get argument data
		d, err := parseCmdData(rd, cmdLen)
		if err != nil {
			Log.Error("tcp:parseCmdData failed() (%s)", err.Error())
			return nil, err
		}

		// append args
		args = append(args, string(d))
	}

	return args, nil
}

// parseCmdSize get the request protocol cmd size
func parseCmdSize(rd *bufio.Reader, prefix uint8) (int, error) {
	// get command size
	cs, err := rd.ReadBytes('\n')
	if err != nil {
		Log.Error("tcp:rd.ReadBytes('\\n') failed (%s)", err.Error())
		return 0, err
	}

	csl := len(cs)
	if csl <= 3 || cs[0] != prefix || cs[csl-2] != '\r' {
		Log.Error("tcp:\"%v\"(%d) number format error, length error or prefix error or no \\r", cs, csl)
		return 0, CmdFmtErr
	}

	// skip the \r\n
	cmdSize, err := strconv.Atoi(string(cs[1 : csl-2]))
	if err != nil {
		Log.Error("tcp:\"%v\" number parse int failed (%s)", cs, err.Error())
		return 0, CmdFmtErr
	}

	return cmdSize, nil
}

// parseCmdData get the sub request protocol cmd data not included \r\n
func parseCmdData(rd *bufio.Reader, cmdLen int) ([]byte, error) {
	d, err := rd.ReadBytes('\n')
	if err != nil {
		Log.Error("tcp:rd.ReadBytes('\\n') failed (%s)", err.Error())
		return nil, err
	}

	dl := len(d)
	// check last \r\n
	if dl != cmdLen+2 || d[dl-2] != '\r' {
		Log.Error("tcp:\"%v\"(%d) number format error, length error or no \\r", d, dl)
		return nil, CmdFmtErr
	}

	// skip last \r\n
	return d[0 : dl-2], nil
}
