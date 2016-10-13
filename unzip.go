package main

import "github.com/itchio/butler/comm"
import "github.com/itchio/wharf/archiver"

func unzip(file string, dir string, resumeFile string) {
	comm.Opf("Extracting zip %s to %s", file, dir)

	settings := archiver.ExtractSettings{
		Consumer:   comm.NewStateConsumer(),
		ResumeFrom: resumeFile,
		OnUncompressedSizeKnown: func(uncompressedSize int64) {
			comm.StartProgressWithTotalBytes(uncompressedSize)
		},
	}

	res, err := archiver.ExtractPath(file, dir, settings)
	comm.EndProgress()

	must(err)
	comm.Logf("Extracted %d dirs, %d files, %d symlinks", res.Dirs, res.Files, res.Symlinks)
}
