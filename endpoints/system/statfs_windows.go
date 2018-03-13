// +build windows

package system

import (
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/runner/syscallex"
)

func StatFS(path string) (*buse.SystemStatFSResult, error) {
	dfs, err := syscallex.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(path))
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.SystemStatFSResult{
		// XXX: cast hygiene
		FreeSize:  int64(dfs.FreeBytesAvailable),
		TotalSize: int64(dfs.TotalNumberOfBytes),
	}
	return res, nil
}
