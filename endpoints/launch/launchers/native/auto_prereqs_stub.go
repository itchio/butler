// +build !windows

package native

import (
	"github.com/itchio/butler/cmd/prereqs"
	"github.com/itchio/butler/endpoints/launch"
)

func handleAutoPrereqs(params *launch.LauncherParams, pc *prereqs.PrereqsContext) ([]string, error) {
	return nil, nil
}
