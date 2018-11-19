package itchio

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// APIError represents an itch.io API error. Some errors
// are just HTTP status codes, others have more detailed messages.
type APIError struct {
	Messages   []string `json:"messages"`
	StatusCode int      `json:"statusCode"`
}

var _ error = (*APIError)(nil)

func (ae *APIError) Error() string {
	return fmt.Sprintf("itch.io API error (%d): %s", ae.StatusCode, strings.Join(ae.Messages, ", "))
}

// IsAPIError returns true if an error is an itch.io API error,
// even if it's wrapped with github.com/pkg/errors
func IsAPIError(err error) bool {
	_, ok := AsAPIError(err)
	return ok
}

// AsAPIError returns an *APIError and true if the
// passed error (no matter how deeply wrapped it is)
// is an *APIError. Otherwise it returns nil, false.
func AsAPIError(err error) (*APIError, bool) {
	rootErr := errors.Cause(err)
	apiError, ok := rootErr.(*APIError)
	return apiError, ok
}
