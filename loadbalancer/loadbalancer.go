package loadbalancer

import (
	"fmt"
	"net"
	"net/url"
)

// Supported protocols.
const (
	ProtocolTCP = "tcp"
	ProtocolUDP = "udp"
)

// A Loadbalancer holds the primitives used to loadbalance the backends of a proxy frontend.
type Loadbalancer interface {
	// Frontend returns the listening address of the proxy.
	Frontend() net.Addr
	// Backend returns the next backend's endpoint on which the data is forwarded to.
	Backend() net.Addr
	// Backends returns all backend's endpoints on which the data can be forwarded to.
	Backends() []net.Addr
}

// ParseDSN returns the loadbalancer parameters extracted from the given DSN.
func ParseDSN(dsn string) (protocol, frontend string, backends []string, err error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", "", nil, err
	}

	return u.Scheme, u.Host, u.Query()["backend"], err
}

// Resolve returns the resolved address of the given parameters.
func Resolve(protocol, address string) (net.Addr, error) {
	switch protocol {
	case ProtocolUDP:
		return net.ResolveUDPAddr("udp", address)
	case ProtocolTCP:
		return net.ResolveTCPAddr("tcp", address)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
