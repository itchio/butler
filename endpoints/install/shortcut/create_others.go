// +build !windows

package shortcut

import "github.com/pkg/errors"

func Create(params CreateParams) error {
	return errors.Errorf("stub")
}
