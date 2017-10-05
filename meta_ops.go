package main

import (
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/tlc"
)

func walk(dir string) {
	startTime := time.Now()

	container, err := tlc.WalkDir(dir, func(fi os.FileInfo) bool { return true })
	must(err)

	totalEntries := 0
	send := func(path string) {
		totalEntries++
		comm.Result(&mansion.WalkResult{
			Type: "entry",
			Path: path,
		})

		comm.Debugf("%s", path)
	}

	for _, f := range container.Files {
		send(f.Path)
	}

	for _, s := range container.Symlinks {
		send(s.Path)
	}

	comm.Result(&mansion.WalkResult{
		Type: "totalSize",
		Size: container.Size,
	})

	comm.Statf("%d entries (%s) walked in %s", totalEntries, humanize.IBytes(uint64(container.Size)), time.Since(startTime))
}
