package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	time.Sleep(time.Nanosecond)
	id := strconv.FormatInt(time.Now().UnixNano(), 36)

	var c net.Conn
	if len(os.Args) == 2 && os.Args[1] == "tcp" {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:7777")
		if err != nil {
			panic(err)
		}

		c, err = net.DialTCP("tcp", nil, addr)
		if err != nil {
			panic(err)
		}
	} else {
		addr, err := net.ResolveUDPAddr("udp", "localhost:7777")
		if err != nil {
			panic(err)
		}

		c, err = net.DialUDP("udp", nil, addr)
		if err != nil {
			panic(err)
		}
	}

	//
	//
	//

	_, err := c.Write([]byte("Coucou " + id))
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 1500)
	n, err := c.Read(buffer)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(buffer[:n]))

	for {
		_, err = c.Write(buffer[:n])
		if err != nil {
			panic(err)
		}

		buffer := make([]byte, 1500)
		n, err := c.Read(buffer)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(buffer[:n]))
		time.Sleep(1 * time.Millisecond)
	}
}
