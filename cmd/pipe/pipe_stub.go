// +build !windows

package pipe

import (
	"fmt"

	"github.com/itchio/butler/mansion"
)

func Do(ctx *mansion.Context, command []string, stdin string, stdout string, stderr string) error {
	return fmt.Errorf("pipe is a windows-only command")
}
