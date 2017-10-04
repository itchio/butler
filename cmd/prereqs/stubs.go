// +build !windows

package prereqs

import (
	"fmt"

	"github.com/itchio/butler/butler"
)

func Test(ctx *butler.Context, prereqs []string) error {
	return fmt.Errorf("test-prereqs is a windows-only command")
}

func Install(ctx *butler.Context, planPath string, pipePath string) error {
	return fmt.Errorf("install-prereqs is a windows-only command")
}
