package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

func main() {
	time.Sleep(time.Nanosecond)
	id := strconv.FormatInt(time.Now().UnixNano(), 36)

	if len(os.Args) < 2 {
		panic("missing url")
	}

	u, err := url.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	var c net.Conn
	if u.Scheme == "tcp" {
		addr, err := net.ResolveTCPAddr(u.Scheme, u.Host)
		if err != nil {
			panic(err)
		}

		c, err = net.DialTCP(u.Scheme, nil, addr)
		if err != nil {
			panic(err)
		}
	} else {
		addr, err := net.ResolveUDPAddr(u.Scheme, u.Host)
		if err != nil {
			panic(err)
		}

		c, err = net.DialUDP(u.Scheme, nil, addr)
		if err != nil {
			panic(err)
		}
	}

	//
	//
	//

	_, err = c.Write([]byte("Coucou " + id))
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
