package proxy

import (
	"context"
	"encoding/binary"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mdouchement/logger"
)

// Imported/Inspired from https://github.com/moby/libnetwork/blob/28576a4038783dfd6f300f83e3076740179ef035/cmd/proxy/udp_proxy.go

const (
	// UDPConnTrackTimeout is the timeout used for UDP connection tracking
	UDPConnTrackTimeout = 90 * time.Second
	// UDPBufSize is the buffer size for the UDP proxy
	UDPBufSize = 65507
)

// UDPProxy is proxy for which handles UDP datagrams. It implements the Proxy
// interface to handle UDP traffic forwarding between the frontend and backend
// addresses.
type UDPProxy struct {
	ctx        context.Context
	listener   *net.UDPConn
	addresser  Addresser
	tracking   connTrackMap
	mutex      sync.Mutex
	acceptable AcceptableConnection
}

// NewUDPProxy creates a new UDPProxy.
func NewUDPProxy(ctx context.Context, addresser Addresser, h AcceptableConnection) (*UDPProxy, error) {
	log := logger.LogWith(ctx)

	// detect version of hostIP to bind only to correct version
	frontend := addresser.Frontend().(*net.UDPAddr)
	fipv := ipv4
	if frontend.IP.To4() == nil {
		fipv = ipv6
	}
	scheme := "udp" + string(fipv)

	backend := addresser.Backend().(*net.UDPAddr)
	bipv := ipv4
	if backend.IP.To4() == nil {
		bipv = ipv6
	}
	log.Infof("Listening on %s://%s forwarded to udp%s://%s", scheme, frontend, bipv, backend)

	listener, err := net.ListenUDP(scheme, frontend)
	if err != nil {
		return nil, err
	}

	return &UDPProxy{
		ctx:        logger.WithLogger(ctx, log.WithPrefixf("[%s://%s]", scheme, frontend)),
		listener:   listener,
		addresser:  addresser,
		tracking:   make(connTrackMap),
		acceptable: h,
	}, nil
}

// FrontendAddr returns the UDP address on which the proxy is listening.
func (p *UDPProxy) FrontendAddr() net.Addr {
	return p.addresser.Frontend()
}

// BackendAddr returns the proxied UDP address.
func (p *UDPProxy) BackendAddr() net.Addr {
	return p.addresser.Backend()
}

// Run starts forwarding the traffic using UDP.
func (p *UDPProxy) Run() {
	log := logger.LogWith(p.ctx)

	buf := make([]byte, UDPBufSize)
	for {
		read, from, err := p.listener.ReadFromUDP(buf)
		if err != nil {
			// NOTE: Apparently ReadFrom doesn't return
			// ECONNREFUSED like Read do (see comment in
			// UDPProxy.replyLoop)
			if !isClosedError(err) {
				log.Warnf("Stopping proxy on udp/%v (%s)", p.addresser.Frontend(), err)
				break
			}

			log.WithError(err).Debugf("Connection closed for %v", from)
			break
		}

		if !p.acceptable(p.ctx, from.IP) {
			continue
		}

		// Handle asynchronously this connection after the first synchronous datagram.
		var proxyConn *net.UDPConn

		next := func() bool { // Use of anonymous function in order to properly defer the unlock
			fromKey := newConnTrackKey(from)
			p.mutex.Lock()
			defer p.mutex.Unlock()

			var hit bool
			proxyConn, hit = p.tracking[fromKey]
			if !hit {
				backend := p.addresser.Backend().(*net.UDPAddr)
				log.Infof("Forwarding %s://%s to %s://%s", p.FrontendAddr().Network(), p.FrontendAddr().String(), backend.Network(), backend.String())

				proxyConn, err = net.DialUDP("udp", nil, backend)
				if err != nil {
					log.Warnf("Can't proxy a datagram to udp/%s: %s\n", backend, err)
					return true
				}

				p.tracking[fromKey] = proxyConn
				go p.replyLoop(proxyConn, from, fromKey)
			}

			return false
		}()

		if next {
			continue
		}

		// Send the datagram synchronously to the backend then replyLoop will handle all the traffic for this connection.
		for i := 0; i != read; {
			written, err := proxyConn.Write(buf[i:read])
			if err != nil {
				log.Warnf("Can't proxy a datagram to udp/%s: %s\n", proxyConn.RemoteAddr().String(), err)
				break
			}

			i += written
		}
	}
}

func (p *UDPProxy) replyLoop(c *net.UDPConn, addr *net.UDPAddr, key connTrackKey) {
	log := logger.LogWith(p.ctx)

	defer func() {
		p.mutex.Lock()
		defer p.mutex.Unlock()

		c, ok := p.tracking[key]
		if ok {
			delete(p.tracking, key)
			c.Close()
		}
	}()

	buf := make([]byte, UDPBufSize)
	reset := true
	for {
		if reset {
			c.SetReadDeadline(time.Now().Add(UDPConnTrackTimeout)) //nolint:errcheck
		}

		read, err := c.Read(buf)
		if err != nil {
			if err, ok := err.(*net.OpError); ok && err.Err == syscall.ECONNREFUSED {
				// This will happen if the last write failed
				// (e.g: nothing is actually listening on the
				// proxied port on the container), ignore it
				// and continue until UDPConnTrackTimeout
				// expires:
				reset = false
				continue
			}

			log.WithError(err).Debugf("Connection closed for %v", addr)
			return
		}

		reset = true

		for i := 0; i != read; {
			written, err := p.listener.WriteToUDP(buf[i:read], addr)
			if err != nil {
				return
			}

			i += written
		}
	}
}

// Close stops forwarding the traffic.
func (p *UDPProxy) Close() {
	p.listener.Close()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, c := range p.tracking {
		c.Close()
	}
}

func isClosedError(err error) bool {
	/* This comparison is ugly, but unfortunately, net.go doesn't export errClosing.
	 * See:
	 * http://golang.org/src/pkg/net/net.go
	 * https://code.google.com/p/go/issues/detail?id=4337
	 * https://groups.google.com/forum/#!msg/golang-nuts/0_aaCvBmOcM/SptmDyX1XJMJ
	 */
	return strings.HasSuffix(err.Error(), "use of closed network connection")
}

//
// Connection tracking.
//

type (
	connTrackMap map[connTrackKey]*net.UDPConn

	// A connTrackKey (net.Addr) where the IP is split into two fields so you can use it as a key in a map.
	connTrackKey struct {
		IPHigh uint64
		IPLow  uint64
		Port   int
	}
)

func newConnTrackKey(addr *net.UDPAddr) connTrackKey {
	if len(addr.IP) == net.IPv4len {
		return connTrackKey{
			IPHigh: 0,
			IPLow:  uint64(binary.BigEndian.Uint32(addr.IP)),
			Port:   addr.Port,
		}
	}

	return connTrackKey{
		IPHigh: binary.BigEndian.Uint64(addr.IP[:8]),
		IPLow:  binary.BigEndian.Uint64(addr.IP[8:]),
		Port:   addr.Port,
	}
}
