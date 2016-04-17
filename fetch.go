package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/archiver"
)

func fetch(spec string, outPath string) {
	must(doFetch(spec, outPath))
}

func doFetch(spec string, outPath string) error {
	var err error

	err = os.MkdirAll(outPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	outFiles, err := ioutil.ReadDir(outPath)
	if err != nil {
		return err
	}

	if len(outFiles) > 0 {
		return fmt.Errorf("Destination directory %s exists and is not empty", outPath)
	}

	target, channel, err := parseSpec(spec)
	if err != nil {
		return err
	}

	client, err := authenticateViaOauth()
	if err != nil {
		return err
	}

	comm.Opf("Getting last build of channel %s", channel)

	channelResponse, err := client.GetChannel(target, channel)
	if err != nil {
		return err
	}

	if channelResponse.Channel.Head == nil {
		return fmt.Errorf("Channel %s doesn't have any builds yet", channel)
	}

	head := *channelResponse.Channel.Head
	var headArchive *itchio.BuildFileInfo

	for _, file := range head.Files {
		comm.Debugf("found file %v", file)
		if file.Type == itchio.BuildFileType_ARCHIVE && file.SubType == itchio.BuildFileSubType_DEFAULT && file.State == itchio.BuildFileState_UPLOADED {
			headArchive = file
			break
		}
	}

	if headArchive == nil {
		return fmt.Errorf("Channel %s's latest build is still processing", channel)
	}

	dlReader, err := client.DownloadBuildFile(head.ID, headArchive.ID)
	if err != nil {
		return err
	}

	tmpFile, err := ioutil.TempFile("", "butler-fetch")
	if err != nil {
		return err
	}

	defer func() {
		if cErr := os.Remove(tmpFile.Name()); err == nil && cErr != nil {
			err = cErr
		}
	}()

	comm.Opf("Downloading build %d", head.ID)

	archiveSize, err := io.Copy(tmpFile, dlReader)
	if err != nil {
		return err
	}

	_, err = tmpFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	comm.Opf("Extracting into %s", outPath)
	result, err := archiver.Extract(tmpFile, archiveSize, outPath, comm.NewStateConsumer())
	if err != nil {
		return err
	}

	comm.Statf("Extracted %d dirs, %d files, %d links into %s", result.Dirs, result.Files, result.Symlinks, outPath)
	return err
}
