package main

import (
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/archiver"
)

func mkdir(dir string) {
	comm.Debugf("mkdir -p %s", dir)

	must(os.MkdirAll(dir, archiver.DirMode))
}

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
