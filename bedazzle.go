package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
)

func bedazzle(spec string) {
	must(doBedazzle(spec))
}

func doBedazzle(spec string) error {
	spec = strings.ToLower(spec)

	target, channel, err := parseSpec(spec)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	client, err := authenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Opf("Querying last build of %s/%s", target, channel)

	channelResponse, err := client.GetChannel(target, channel)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if channelResponse.Channel.Head == nil {
		return fmt.Errorf("Channel %s doesn't have any builds yet", channel)
	}

	head := *channelResponse.Channel.Head

	patchPath := "bedazzle.pwr"
	patchWriter, err := os.Create(patchPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sigPath := "bedazzle.pws"
	sigWriter, err := os.Create(sigPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	writers := map[itchio.BuildFileType]io.WriteCloser{
		itchio.BuildFileType_PATCH:     patchWriter,
		itchio.BuildFileType_SIGNATURE: sigWriter,
	}

	done := make(chan bool)
	errs := make(chan error)

	comm.Opf("Downloading patch and signature...")
	comm.StartProgress()

	for _, f := range head.Files {
		writer := writers[f.Type]
		if writer != nil {
			go func(f *itchio.BuildFileInfo) {
				reader, err := client.DownloadBuildFile(head.ID, f.ID)
				if err != nil {
					errs <- err
					return
				}

				writer = counter.NewWriterCallback(func(count int64) {
					comm.Progress(float64(count) / float64(f.Size))
				}, writer)

				_, err = io.Copy(writer, reader)
				if err != nil {
					errs <- err
					return
				}

				err = writer.Close()
				if err != nil {
					errs <- err
					return
				}

				done <- true
			}(f)
		}
	}

	for i := 0; i < 2; i++ {
		select {
		case err = <-errs:
			return errors.Wrap(err, 1)
		case <-done:
			// good!
		}
	}

	comm.EndProgress()

	parent, err := client.ListBuildFiles(head.ParentBuildID)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	archivePath := "bedazzle.zip"
	archiveWriter, err := os.Create(archivePath)

	foundArchive := false

	comm.Opf("Downloading archive...")
	comm.StartProgress()

	for _, f := range parent.Files {
		writer := archiveWriter

		if f.Type == itchio.BuildFileType_ARCHIVE {
			foundArchive = true

			go func(f itchio.BuildFileInfo) {
				reader, err := client.DownloadBuildFile(head.ParentBuildID, f.ID)
				if err != nil {
					errs <- err
					return
				}

				cw := counter.NewWriterCallback(func(count int64) {
					comm.Progress(float64(count) / float64(f.Size))
				}, archiveWriter)

				_, err = io.Copy(cw, reader)
				if err != nil {
					errs <- err
					return
				}

				err = writer.Close()
				if err != nil {
					errs <- err
					return
				}

				done <- true
			}(f)
		}
	}

	if !foundArchive {
		return fmt.Errorf("no archive found in parent build %d", head.ParentBuildID)
	}

	select {
	case err = <-errs:
		return errors.Wrap(err, 1)
	case <-done:
		// all good!
	}

	comm.EndProgress()

	comm.Opf("Wiping existing block store")
	wipe("blocks")
	wipe("outblocks")

	err = os.MkdirAll("outblocks", 0755)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	manifestPath := "bedazzle-old.pwm"

	comm.Opf("Splitting old archive")
	split(archivePath, manifestPath)

	newManifestPath := "bedazzle-new.pwm"

	comm.Opf("Applying patch with filters & 0 latency")
	*rangesArgs.infilter = true
	*rangesArgs.outfilter = true
	*rangesArgs.inlatency = 0
	*rangesArgs.outlatency = 0
	*rangesArgs.fanout = runtime.NumCPU() + 1
	ranges(manifestPath, patchPath, newManifestPath)

	comm.Opf("Merging outblocks with blocks")
	ditto("outblocks", "blocks")

	outPath := "bedazzle-out"

	comm.Opf("Unsplitting with new manifest")
	unsplit(outPath, newManifestPath)

	comm.Opf("Checking against signature")
	verify(sigPath, outPath)

	comm.Statf("Success!")

	return nil
}
