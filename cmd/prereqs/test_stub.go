// +build !windows

package prereqs

import (
	"fmt"

	"github.com/itchio/butler/mansion"
)

func Test(ctx *mansion.Context, prereqs []string) error {
	return fmt.Errorf("test-prereqs is a windows-only command")
}
