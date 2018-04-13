// The timeout package provides an http.Client that closes a connection if it takes
// too long to establish, or stays idle for too long.
package timeout

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/efarrer/iothrottler"
	"github.com/getlantern/idletiming"
	"github.com/pkg/errors"
)

var IgnoreCertificateErrors = os.Getenv("HTTPKIT_IGNORE_CERTIFICATE_ERRORS") == "1"

const (
	DefaultConnectTimeout time.Duration = 30 * time.Second
	DefaultIdleTimeout                  = 60 * time.Second
)

var ThrottlerPool *iothrottler.IOThrottlerPool

func init() {
	ThrottlerPool = iothrottler.NewIOThrottlerPool(iothrottler.Unlimited)
}

var simulateOffline bool = false
var conns = make(map[net.Conn]bool)

func SetSimulateOffline(enabled bool) {
	simulateOffline = enabled
}

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (net.Conn, error) {
	return func(netw, addr string) (net.Conn, error) {
		if simulateOffline {
			return nil, &net.OpError{
				Op:  "dial",
				Err: errors.New("simulated offline"),
			}
		}

		// if it takes too long to establish a connection, give up
		timeoutConn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// respect global throttle settings
		throttledConn, err := ThrottlerPool.AddConn(timeoutConn)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// measure bps
		monitorConn := &monitoringConn{
			Conn: throttledConn,
		}
		// if we stay idle too long, close
		idleConn := idletiming.Conn(monitorConn, rwTimeout, func() {
			monitorConn.Close()
		})
		return idleConn, nil
	}
}

func NewClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial:  timeoutDialer(connectTimeout, readWriteTimeout),
	}
	if IgnoreCertificateErrors {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return &http.Client{
		Transport: transport,
	}
}

func NewDefaultClient() *http.Client {
	return NewClient(DefaultConnectTimeout, DefaultIdleTimeout)
}
