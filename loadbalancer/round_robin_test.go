package loadbalancer_test

import (
	"net/url"
	"testing"

	"github.com/mdouchement/geoblock-proxy/loadbalancer"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobinAsLoadbalancer(t *testing.T) {
	var lb loadbalancer.Loadbalancer
	var err error

	lb, err = loadbalancer.NewRoundRobin("udp://localhost:5050?backend=localhost:5000")
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:5050", lb.Frontend().String())
	assert.Len(t, lb.Backends(), 1)
	for _, backend := range lb.Backends() {
		assert.Equal(t, "127.0.0.1:5000", backend.String())
	}
	assert.Equal(t, "127.0.0.1:5000", lb.Backend().String())
	assert.Equal(t, "127.0.0.1:5000", lb.Backend().String())
}

func TestRoundRobin_Backend(t *testing.T) {
	q := url.Values{
		"backend": []string{
			"127.0.0.1:5000",
			"127.0.0.1:5001",
			"127.0.0.1:5002",
		},
	}
	dsn := url.URL{
		Scheme:   "udp",
		Host:     "127.0.0.1:5050",
		RawQuery: q.Encode(),
	}

	n := len(q["backend"])

	//

	lb, err := loadbalancer.NewRoundRobin(dsn.String())
	assert.NoError(t, err)
	for i := 0; i < 100*n; i++ {
		assert.Equal(t, q["backend"][i%n], lb.Backend().String())
	}
}
