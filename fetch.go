package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/archiver"
)

func fetch(specStr string, outPath string) {
	must(doFetch(specStr, outPath))
}

func doFetch(specStr string, outPath string) error {
	var err error

	err = os.MkdirAll(outPath, os.FileMode(0755))
	if err != nil {
		return errors.Wrap(err, 1)
	}

	outFiles, err := ioutil.ReadDir(outPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if len(outFiles) > 0 {
		return fmt.Errorf("Destination directory %s exists and is not empty", outPath)
	}

	spec, err := itchio.ParseSpec(specStr)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = spec.EnsureChannel()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	client, err := authenticateViaOauth()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Opf("Getting last build of channel %s", spec.Channel)

	channelResponse, err := client.GetChannel(spec.Target, spec.Channel)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	if channelResponse.Channel.Head == nil {
		return fmt.Errorf("Channel %s doesn't have any builds yet", spec.Channel)
	}

	head := *channelResponse.Channel.Head
	var headArchive *itchio.BuildFile

	for _, file := range head.Files {
		comm.Debugf("found file %v", file)
		if file.Type == itchio.BuildFileTypeArchive && file.SubType == itchio.BuildFileSubTypeDefault && file.State == itchio.BuildFileStateUploaded {
			headArchive = file
			break
		}
	}

	if headArchive == nil {
		return fmt.Errorf("Channel %s's latest build is still processing", spec.Channel)
	}

	dlReader, err := client.DownloadBuildFile(head.ID, headArchive.ID)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	tmpFile, err := ioutil.TempFile("", "butler-fetch")
	if err != nil {
		return errors.Wrap(err, 1)
	}

	defer func() {
		if cErr := os.Remove(tmpFile.Name()); err == nil && cErr != nil {
			err = cErr
		}
	}()

	comm.Opf("Downloading build %d", head.ID)

	archiveSize, err := io.Copy(tmpFile, dlReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	_, err = tmpFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	settings := archiver.ExtractSettings{
		Consumer: comm.NewStateConsumer(),
	}

	comm.Opf("Extracting into %s", outPath)
	result, err := archiver.Extract(tmpFile, archiveSize, outPath, settings)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Statf("Extracted %d dirs, %d files, %d links into %s", result.Dirs, result.Files, result.Symlinks, outPath)

	if err != nil {
		return errors.Wrap(err, 1)
	}
	return nil
}
