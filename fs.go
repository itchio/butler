package main

import (
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
)

func sizeof(path string) {
	totalSize := int64(0)

	inc := func(_ string, f os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		totalSize += f.Size()
		return nil
	}

	filepath.Walk(path, inc)
	comm.Logf("Total size of %s: %s", path, humanize.IBytes(uint64(totalSize)))
	comm.Result(totalSize)
}
