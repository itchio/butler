package wtest

import (
	"testing"

	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
)

// Must shows a complete error stack and fails a test immediately
// if err is non-nil
func Must(t *testing.T, err error) {
	if err != nil {
		if se, ok := err.(*errors.Error); ok {
			t.Logf("Full stack: %s", se.ErrorStack())
		}
		assert.NoError(t, err)
		t.FailNow()
	}
}
