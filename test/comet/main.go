package main

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"time"
)

func main() {
	cmd := "sub"
	key := "Terry-Mao"
	mid := 0
	heartbeat := 10

	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		panic(err)
	}

	fmt.Println("connect to gopush2")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		panic(err)
	}

	fmt.Println("send sub request")
	if _, err := conn.Write([]byte(fmt.Sprintf("*4\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n$%d\r\n10\r\n", len(cmd), cmd, len(key), key, len(strconv.Itoa(mid)), mid, len(strconv.Itoa(heartbeat)), heartbeat))); err != nil {
		panic(err)
	}

	buf := make([]byte, 1024)
	// get first heartbeat
	n, err := conn.Read(buf)
	if err != nil {
		panic(err)
	}

	bufStr := string(buf[0:n])
	if bufStr == "h" {
		fmt.Println("get first heartbeat, start goroutine to sending heartbeat period")
	} else {
		panic("unknown heartbeat protocol")
	}

	// send heartbeat
	go func() {
		fmt.Println("send heartbeat")
		for {
			if _, err := conn.Write([]byte("h")); err != nil {
				panic(err)
			}

			time.Sleep(time.Duration(heartbeat) * time.Second)
		}
	}()

	// block read reply from service
	fmt.Println("wait message")
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat*2))); err != nil {
			panic(err)
		}

		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}

		bufStr := string(buf[0:n])
		if bufStr != "h" {
			tbuf := bytes.NewBuffer(buf)
			line, err := tbuf.ReadBytes('\n')
			if err != nil {
				panic(err)
			}

			if len(line) < 3 || line[0] != '$' || line[len(line)-2] != '\r' {
				panic("protocol format1 error")
			}

			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				panic(err)
			}

			data := make([]byte, cmdSize+2)
			n, err := tbuf.Read(data)
			if err != nil {
				panic(err)
			}

			if n != cmdSize+2 {
				panic("protocol size error")
			}

			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				panic("protocol format2 error")
			}

			fmt.Println(string(data[0:cmdSize]))
		} else if bufStr == "h" {
			// receive heartbeat continue
			// fmt.Println("heartbeat")
			continue
		} else {
			panic("unknow protocol")
		}
	}
}
