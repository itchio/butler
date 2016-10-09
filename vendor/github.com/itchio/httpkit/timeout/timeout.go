// The timeout package provides an http.Client that closes a connection if it takes
// too long to establish, or stays idle for too long.
package timeout

import (
	"net"
	"net/http"
	"time"

	"github.com/getlantern/idletiming"
	"github.com/go-errors/errors"
)

const (
	DefaultConnectTimeout time.Duration = 30 * time.Second
	DefaultIdleTimeout                  = 60 * time.Second
)

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (net.Conn, error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
		idleConn := idletiming.Conn(conn, rwTimeout, func() {
			conn.Close()
		})
		return idleConn, nil
	}
}

func NewClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial:  timeoutDialer(connectTimeout, readWriteTimeout),
		},
	}
}

func NewDefaultClient() *http.Client {
	return NewClient(DefaultConnectTimeout, DefaultIdleTimeout)
}
