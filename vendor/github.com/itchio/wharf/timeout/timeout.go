package timeout

import (
	"net"
	"net/http"
	"time"

	"github.com/getlantern/idletiming"
	"github.com/itchio/butler/comm"
)

const (
	DefaultConnectTimeout time.Duration = 30 * time.Second
	DefaultIdleTimeout                  = 60 * time.Second
)

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		idleConn := idletiming.Conn(conn, rwTimeout, func() {
			comm.Logf("connection was idle for too long, dropping")
			conn.Close()
		})
		return idleConn, nil
	}
}

func NewClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: timeoutDialer(connectTimeout, readWriteTimeout),
		},
	}
}

func NewDefaultClient() *http.Client {
	return NewClient(DefaultConnectTimeout, DefaultIdleTimeout)
}
