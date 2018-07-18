package itchio

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

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

func AsAPIError(err error) (error, bool) {
	rootErr := errors.Cause(err)
	apiError, ok := rootErr.(*APIError)
	return apiError, ok
}
