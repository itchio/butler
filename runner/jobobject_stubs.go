// +build !windows

package runner

import "github.com/itchio/wharf/state"

func SetupJobObject(consumer *state.Consumer) error {
	// nothing to do
	return nil
}

func WaitJobObject(consumer *state.Consumer) error {
	// nothing to do
	return nil
}
