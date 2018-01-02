// +build !windows

package elevate

import (
	"fmt"
)

func Elevate(params *ElevateParams) (int, error) {
	return 0, fmt.Errorf("elevate is a windows-only command")
}
