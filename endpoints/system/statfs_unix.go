//go:build !windows
// +build !windows

package system

import (
	"syscall"

	"github.com/itchio/butler/butlerd"
	"github.com/pkg/errors"
)

func StatFS(path string) (*butlerd.SystemStatFSResult, error) {
	var stats syscall.Statfs_t
	err := syscall.Statfs(path, &stats)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// XXX: bad cast hygiene. For very, very, very, very
	// large volumes, this will overflow.
	var freeSize int64 = int64(stats.Bavail) * int64(stats.Bsize)
	var totalSize int64 = int64(stats.Blocks) * int64(stats.Bsize)

	res := &butlerd.SystemStatFSResult{
		FreeSize:  freeSize,
		TotalSize: totalSize,
	}
	return res, nil
}
