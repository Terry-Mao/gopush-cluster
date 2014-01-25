package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"time"
)

func main() {
	cmd := "sub"
	key := "Terry-Mao"
	heartbeat := 30

	addr, err := net.ResolveTCPAddr("tcp", "10.33.13.175:6969")
	if err != nil {
		panic(err)
	}

	fmt.Println("connect to gopush2")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		panic(err)
	}

	fmt.Println("send sub request")
	proto := []byte(fmt.Sprintf("*3\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n", len(cmd), cmd, len(key), key, len(strconv.Itoa(heartbeat)), heartbeat))
	fmt.Println(string(proto))
	if _, err := conn.Write(proto); err != nil {
		panic(err)
	}

	// get first heartbeat
	first := false
	rd := bufio.NewReader(conn)
	// block read reply from service
	fmt.Println("wait message")
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(heartbeat*2))); err != nil {
			panic(err)
		}

		line, err := rd.ReadBytes('\n')
		if err != nil {
			fmt.Printf("(%s) %v", string(line), line)
			panic(err)
		}

		if line[len(line)-2] != '\r' {
			panic("protocol format1 error")
		}

		switch line[0] {
		// reply
		case '$':
			cmdSize, err := strconv.Atoi(string(line[1 : len(line)-2]))
			if err != nil {
				panic(err)
			}

			data, err := rd.ReadBytes('\n')
			if err != nil {
				panic(err)
			}

			if len(data) != cmdSize+2 {
				panic("protocol size error")
			}

			if data[cmdSize] != '\r' || data[cmdSize+1] != '\n' {
				panic("protocol format2 error")
			}

			reply := string(data[0:cmdSize])
			fmt.Println(reply)
			break
			// heartbeat
		case '+':
			if !first {
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

				first = true
			}
			fmt.Println("heartbeat")
			break
		}
	}
}
