// +build !windows

package native

import "github.com/itchio/butler/endpoints/launch"

func handleUE4Prereqs(params launch.LauncherParams) error {
	// nothing to worry about on non-windows platforms
	return nil
}
