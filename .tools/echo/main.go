package main

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"sync/atomic"
)

func main() {
	if len(os.Args) < 2 {
		panic("missing url")
	}

	u, err := url.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	if u.Scheme == "tcp" {
		tcp(u)
	}

	udp(u)
}

func udp(url *url.URL) {
	addr, err := net.ResolveUDPAddr(url.Scheme, url.Host)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listen on udp://%s\n", addr)
	c, err := net.ListenUDP(url.Scheme, addr)
	if err != nil {
		panic(err)
	}

	var i uint64

	for {
		i++

		buffer := make([]byte, 1500)
		n, addr, err := c.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("ERROR:", err)
			continue
		}

		fmt.Println(i, string(buffer[:n]))

		_, err = c.WriteToUDP(buffer[:n], addr)
		if err != nil {
			fmt.Println("ERROR:", err)
		}
	}
}

func tcp(url *url.URL) {
	addr, err := net.ResolveTCPAddr(url.Scheme, url.Host)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listen on tcp://%s\n", addr)
	l, err := net.ListenTCP(url.Scheme, addr)
	if err != nil {
		panic(err)
	}

	var i uint64

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("ERROR:", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()

			for {
				j := atomic.AddUint64(&i, 1)

				buffer := make([]byte, 1500)
				n, err := c.Read(buffer)
				if err != nil {
					if err == io.EOF {
						return
					}

					fmt.Println("ERROR:", err)
					return
				}

				fmt.Println(j, string(buffer[:n]))

				_, err = c.Write(buffer[:n])
				if err != nil {
					fmt.Println("ERROR:", err)
				}
			}
		}(c)
	}
}
