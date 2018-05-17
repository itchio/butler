package wtest

import (
	"testing"
)

// Must shows a complete error stack and fails a test immediately
// if err is non-nil
func Must(t *testing.T, err error) {
	if err != nil {
		t.Helper()
		t.Errorf("%+v", err)
		t.FailNow()
	}
}
