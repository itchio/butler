package itchio

import (
	"fmt"
	"strings"
)

type APIError struct {
	Messages   []string
	StatusCode int
}

var _ error = (*APIError)(nil)

func (ae *APIError) Error() string {
	return fmt.Sprintf("itch.io API error (%d): %s", ae.StatusCode, strings.Join(ae.Messages, ", "))
}

type causer interface {
	Cause() error
}

// IsApiError returns true if an error is an itch.io API error,
// even if it's wrapped with github.com/pkg/errors
func IsAPIError(err error) bool {
	if err == nil {
		return false
	}

	if se, ok := err.(causer); ok {
		return IsAPIError(se.Cause())
	}

	_, ok := err.(*APIError)
	return ok
}
