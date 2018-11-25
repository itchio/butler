package neterr

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/getlantern/idletiming"
)

// IsNetworkError returns true if the error's cause is: io.ErrUnexpectedEOF,
// any *net.OpError, any *url.Error, any URL that implements `Temporary()`
// (and returns true)
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if err == io.ErrUnexpectedEOF {
		return true
	}

	if causer, ok := err.(causer); ok {
		return IsNetworkError(causer.Cause())
	}

	if urlError, ok := err.(*url.Error); ok {
		return IsNetworkError(urlError.Err)
	}

	if _, ok := err.(*net.OpError); ok {
		return true
	}

	if err == idletiming.ErrIdled {
		return true
	}

	{
		// net/http's http2 errors are unexported structs, I don't know
		// of a better way to detect this :(
		// see net/http/h2_bundle.go
		msg := fmt.Sprintf("%v", err)
		if strings.HasPrefix(msg, "stream error: stream ID ") {
			return true
		}
		if strings.HasPrefix(msg, "connection error: ") {
			return true
		}
	}

	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}

	return false
}

type temporary interface {
	Temporary() bool
}

type causer interface {
	Cause() error
}
