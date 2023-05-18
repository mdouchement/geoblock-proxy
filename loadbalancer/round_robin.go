package loadbalancer

import (
	"fmt"
	"net"
	"sync/atomic"
)

// A RoundRobin is a loadbalancer which walks through the available backends one at a time.
type RoundRobin struct {
	frontend net.Addr
	backends []net.Addr
	index    int32
}

// NewRoundRobin returns a new RoundRobin.
func NewRoundRobin(dsn string) (*RoundRobin, error) {
	protocol, frontend, backends, err := ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	lb := &RoundRobin{
		index:    -1,
		backends: make([]net.Addr, len(backends)),
	}

	lb.frontend, err = Resolve(protocol, frontend)
	if err != nil {
		return nil, fmt.Errorf("frontend: %w", err)
	}

	for i, backend := range backends {
		lb.backends[i], err = Resolve(protocol, backend)
		if err != nil {
			return nil, fmt.Errorf("backend: %w", err)
		}
	}

	return lb, nil
}

// Frontend returns the listening address of the proxy.
func (l *RoundRobin) Frontend() net.Addr {
	return l.frontend
}

// Backend returns the next backend's endpoint on which the data is forwarded to.
func (l *RoundRobin) Backend() net.Addr {
	index := atomic.AddInt32(&l.index, 1)
	return l.backends[int(index)%len(l.backends)]
}

// Backends returns all backend's endpoints on which the data can be forwarded to.
func (l *RoundRobin) Backends() []net.Addr {
	return l.backends
}
