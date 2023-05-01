package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "tcp" {
		tcp()
	}

	udp()
}

func udp() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:7778")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listen on udp://%s\n", addr)
	c, err := net.ListenUDP("udp", addr)
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

func tcp() {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:7778")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Listen on tcp://%s\n", addr)
	l, err := net.ListenTCP("tcp", addr)
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
