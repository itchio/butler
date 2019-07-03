// +build !windows

package prereqs

import "github.com/itchio/headway/state"

func RegistryKeyExists(consumer *state.Consumer, path string) bool {
	return false
}
