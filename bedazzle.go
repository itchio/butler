package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/httpfile"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pools/blockpool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
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

	comm.Logf("")
	comm.Opf("Downloading patch and signature for build #%d...", head.ID)
	comm.StartProgress()

	startTime := time.Now()
	dlSize := int64(0)

	for _, f := range head.Files {
		writer := writers[f.Type]
		if writer != nil {
			go func(f *itchio.BuildFileInfo) {
				dlSize += f.Size

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

	patchDlDuration := time.Since(startTime)
	perSec := float64(dlSize) / patchDlDuration.Seconds()
	comm.Statf("Downloaded %s in %s (%s/s)", humanize.IBytes(uint64(dlSize)), patchDlDuration, humanize.IBytes(uint64(perSec)))

	parent, err := client.ListBuildFiles(head.ParentBuildID)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	var archiveID int64 = -1

	comm.Logf("")
	comm.Opf("Wiping existing block store")
	wipe("blocks")

	err = os.MkdirAll("blocks", 0755)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Logf("")
	comm.Opf("Splitting archive for build #%d...", head.ParentBuildID)

	for _, f := range parent.Files {
		if f.Type == itchio.BuildFileType_ARCHIVE {
			archiveID = f.ID
		}
	}

	if archiveID == -1 {
		return fmt.Errorf("no archive found in parent build %d", head.ParentBuildID)
	}

	getArchiveURL := func() (string, error) {
		r, apiErr := client.GetBuildFileDownloadURL(head.ParentBuildID, archiveID)
		if apiErr != nil {
			return "", errors.Wrap(apiErr, 1)
		}

		return r.URL, nil
	}

	// now comes the real fun part
	hf, err := httpfile.New(getArchiveURL, http.DefaultClient)
	if err != nil {
		return err
	}

	if *bedazzleArgs.debughttpfile {
		hf.Consumer = comm.NewStateConsumer()
	}

	zr, err := zip.NewReader(hf, hf.Size())
	if err != nil {
		return err
	}

	container, err := tlc.WalkZip(zr, filterPaths)
	if err != nil {
		return err
	}

	comm.Statf("Working from remote zip, containing %s in %s", humanize.IBytes(uint64(container.Size)), container.Stats())

	inPool := zippool.New(container, zr)
	manifestPath := "bedazzle-old.pwm"

	err = doSplitCustom(inPool, container, manifestPath)
	if err != nil {
		return err
	}

	newManifestPath := "bedazzle-new.pwm"

	comm.Logf("")
	comm.Opf("Parsing signature...")
	// verify

	sigReader, err := os.Open(sigPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sigContainer, sigHashes, err := pwr.ReadSignature(sigReader)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	sigInfo := &blockpool.SignatureInfo{
		Container: sigContainer,
		Hashes:    sigHashes,
	}

	comm.Logf("")
	comm.Opf("Applying patch with filters, 0 latency, and validating")

	*rangesArgs.infilter = true
	*rangesArgs.outfilter = true
	*rangesArgs.fanout = runtime.NumCPU() + 1

	*rangesArgs.inlatency = 0
	*rangesArgs.outlatency = 0
	err = doRanges(manifestPath, patchPath, newManifestPath, sigInfo)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.Statf("Success!")

	return nil
}
