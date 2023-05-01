package proxy

import (
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/mdouchement/logger"
	"github.com/pkg/errors"
)

// TCPProxy is a proxy for TCP connections. It implements the Proxy interface to
// handle TCP traffic forwarding between the frontend and backend addresses.
type TCPProxy struct {
	ctx        context.Context
	listener   *net.TCPListener
	frontend   *net.TCPAddr
	backend    *net.TCPAddr
	acceptable AcceptableConnection
}

// NewTCPProxy creates a new TCPProxy.
func NewTCPProxy(ctx context.Context, frontend, backend *net.TCPAddr, h AcceptableConnection) (*TCPProxy, error) {
	log := logger.LogWith(ctx)

	// detect version of hostIP to bind only to correct version
	fipv := ipv4
	if frontend.IP.To4() == nil {
		fipv = ipv6
	}
	scheme := "tcp" + string(fipv)

	bipv := ipv4
	if backend.IP.To4() == nil {
		bipv = ipv6
	}
	log.Infof("Listening on %s://%s forwarded to tcp%s://%s", scheme, frontend, bipv, backend)

	listener, err := net.ListenTCP(scheme, frontend)
	if err != nil {
		return nil, err
	}

	// If the port in frontend was 0 then ListenTCP will have a picked
	// a port to listen on, hence the call to Addr to get that actual port:
	return &TCPProxy{
		ctx:        ctx,
		listener:   listener,
		frontend:   listener.Addr().(*net.TCPAddr),
		backend:    backend,
		acceptable: h,
	}, nil
}

// FrontendAddr returns the TCP address on which the proxy is listening.
func (p *TCPProxy) FrontendAddr() net.Addr {
	return p.frontend
}

// BackendAddr returns the proxied TCP address.
func (p *TCPProxy) BackendAddr() net.Addr {
	return p.backend
}

// Run starts forwarding the traffic using TCP.
func (p *TCPProxy) Run() {
	log := logger.LogWith(p.ctx)

	for {
		c, err := p.listener.Accept()
		if err != nil {
			log.Errorf("Could not accept %s", err)
			continue
		}
		// c.(*net.TCPConn).SetKeepAlive(true)

		if !p.acceptable(p.ctx, c.RemoteAddr().(*net.TCPAddr).IP) {
			c.Close()
			continue
		}

		go func(local net.Conn) {
			remote, err := net.DialTCP("tcp", nil, p.backend)
			if err != nil {
				log.Errorf("Could not connect to backend: %s", err)
				return
			}
			// remote.SetKeepAlive(true)

			err = p.relay(local, remote)
			if err != nil && !IsIgnorableError(err) {
				log.Errorf("Could not pipe the TCP connection: %s", err)
			}

			log.WithError(err).Debugf("Connection closed for %v", local.RemoteAddr())
		}(c)
	}
}

func (p *TCPProxy) relay(local, remote net.Conn) error {
	defer local.Close()
	defer remote.Close()

	var err, err1 error
	var wg sync.WaitGroup
	const delay = time.Second

	wg.Add(1)
	go func() {
		defer wg.Done()

		_, err1 = io.Copy(remote, local)
		//nolint:errcheck
		remote.SetDeadline(time.Now().Add(delay)) // wake up the other goroutine blocking on remote
	}()

	_, err = io.Copy(local, remote)
	//nolint:errcheck
	local.SetDeadline(time.Now().Add(delay)) // wake up the other goroutine blocking on local

	wg.Wait()

	if err1 != nil {
		return err1
	}
	return err
}

// Close stops forwarding the traffic.
func (p *TCPProxy) Close() {
	p.listener.Close()
}

// IsIgnorableError returns true if the net error is ignorable.
func IsIgnorableError(err error) bool {
	err = errors.Cause(err)

	ok := strings.HasSuffix(err.Error(), "no such host") ||
		strings.HasSuffix(err.Error(), "connection reset by peer") ||
		strings.HasSuffix(err.Error(), "connection refused")
	if ok {
		return ok
	}

	if err, ok := err.(*net.OpError); ok {
		return err.Timeout()
	}

	if err, ok := err.(net.Error); ok {
		return err.Timeout()
	}

	return false
}
