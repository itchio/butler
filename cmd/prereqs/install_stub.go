// +build !windows

package prereqs

import (
	"fmt"

	"github.com/itchio/butler/mansion"
)

func Install(ctx *mansion.Context, planPath string, pipePath string) error {
	return fmt.Errorf("install-prereqs is a windows-only command")
}
