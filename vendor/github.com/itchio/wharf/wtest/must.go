package wtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Must shows a complete error stack and fails a test immediately
// if err is non-nil
func Must(t *testing.T, err error) {
	if err != nil {
		assert.NoError(t, err)
		t.FailNow()
	}
}
