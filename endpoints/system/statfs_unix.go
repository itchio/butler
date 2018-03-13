// +build !windows

package system

import (
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
)

func StatFS(path string) (*buse.SystemStatFSResult, error) {
	var stats syscall.Statfs_t
	err := syscall.Statfs(path, &stats)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// XXX: bad cast hygiene. For very, very, very, very
	// large volumes, this will overflow.
	var freeSize int64 = int64(stats.Bavail) * stats.Bsize
	var totalSize int64 = int64(stats.Blocks) * stats.Bsize

	res := &buse.SystemStatFSResult{
		FreeSize:  freeSize,
		TotalSize: totalSize,
	}
	return res, nil
}
