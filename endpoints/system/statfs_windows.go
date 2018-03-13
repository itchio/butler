// +build windows

package system

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
)

func StatFS(path string) (*buse.SystemStatFSResult, error) {
	return nil, errors.Errorf("StatFS on windows: stub")
}
