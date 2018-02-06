// +build !windows

package prereqs

import "github.com/itchio/wharf/state"

func RegistryKeyExists(consumer *state.Consumer, path string) bool {
	return false
}
