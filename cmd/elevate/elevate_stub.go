// +build !windows

package elevate

import (
	"fmt"
)

func Do(command []string) error {
	return fmt.Errorf("elevate is a windows-only command")
}
