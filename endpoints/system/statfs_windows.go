// +build windows

package system

import (
	"syscall"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/ox/syscallex"
	"github.com/pkg/errors"
)

func StatFS(path string) (*butlerd.SystemStatFSResult, error) {
	dfs, err := syscallex.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(path))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.SystemStatFSResult{
		// XXX: cast hygiene
		FreeSize:  int64(dfs.FreeBytesAvailable),
		TotalSize: int64(dfs.TotalNumberOfBytes),
	}
	return res, nil
}
