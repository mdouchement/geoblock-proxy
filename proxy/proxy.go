package proxy

import (
	"context"
	"net"
)

// Imported/Inspired from https://github.com/moby/libnetwork/blob/28576a4038783dfd6f300f83e3076740179ef035/cmd/proxy/udp_proxy.go

// ipVersion refers to IP version - v4 or v6
type ipVersion string

const (
	// IPv4 is version 4
	ipv4 ipVersion = "4"
	// IPv4 is version 6
	ipv6 ipVersion = "6"
)

// AcceptableConnection is called when a proxy got a new connection.
// When the handler returns false, the connection is closed.
type AcceptableConnection func(ctx context.Context, ip net.IP) bool

// Proxy defines the behavior of a proxy. It forwards traffic back and forth
// between two endpoints : the frontend and the backend.
// It can be used to do software port-mapping between two addresses.
// e.g. forward all traffic between the frontend (host) 127.0.0.1:3000
// to the backend (container) at 172.17.42.108:4000.
type Proxy interface {
	// Run starts forwarding traffic back and forth between the front
	// and back-end addresses.
	Run()
	// Close stops forwarding traffic and close both ends of the Proxy.
	Close()
	// FrontendAddr returns the address on which the proxy is listening.
	FrontendAddr() net.Addr
	// BackendAddr returns the proxied address.
	BackendAddr() net.Addr
}

// NewProxy creates a Proxy according to the specified frontend and backend.
func NewProxy(ctx context.Context, frontend, backend net.Addr, h AcceptableConnection) (Proxy, error) {
	switch frontend.(type) {
	case *net.UDPAddr:
		return NewUDPProxy(ctx, frontend.(*net.UDPAddr), backend.(*net.UDPAddr), h)
	case *net.TCPAddr:
		return NewTCPProxy(ctx, frontend.(*net.TCPAddr), backend.(*net.TCPAddr), h)
	// case *sctp.SCTPAddr:
	// 	return NewSCTPProxy(frontend.(*sctp.SCTPAddr), backend.(*sctp.SCTPAddr), h)
	default:
		panic("Unsupported protocol")
	}
}
