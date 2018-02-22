package itchio

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
)

type APIError struct {
	Messages []string
}

var _ error = (*APIError)(nil)

func (ae *APIError) Error() string {
	return fmt.Sprintf("itch.io API error: %s", strings.Join(ae.Messages, ", "))
}

// IsApiError returns true if an error is an itch.io API error,
// even if it's wrapped with github.com/go-errors/errors
func IsAPIError(err error) bool {
	if err == nil {
		return false
	}

	if se, ok := err.(*errors.Error); ok {
		return IsAPIError(se.Err)
	}

	_, ok := err.(*APIError)
	return ok
}
