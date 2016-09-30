package main

import "github.com/itchio/butler/comm"
import "github.com/itchio/wharf/archiver"

func unzip(file string, dir string) {
	comm.Opf("Extracting zip %s to %s", file, dir)

	comm.StartProgress()
	res, err := archiver.ExtractPath(file, dir, comm.NewStateConsumer())
	comm.EndProgress()

	must(err)
	comm.Logf("Extracted %d dirs, %d files, %d symlinks", res.Dirs, res.Files, res.Symlinks)
}
