package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/itchio/arkive/zip"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"

	"github.com/pkg/errors"
)

func doZip() error {
	if len(os.Args) < 3 {
		must(errors.Errorf("Usage: htfsmonkey zip [.zip file]"))
	}
	zipPath := os.Args[2]

	zipBytes, err := ioutil.ReadFile(zipPath)
	must(err)
	log.Printf("Read %s zip file", progress.FormatBytes(int64(len(zipBytes))))
	log.Printf("Validating...")

	numFiles := 0
	func() {
		zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
		must(err)

		numFiles = len(zr.File)
		log.Printf("Found %d files in zip", numFiles)

		for _, f := range zr.File {
			rc, err := f.Open()
			must(err)

			_, err = ioutil.ReadAll(rc)
			must(err)

			must(rc.Close())
		}
	}()
	log.Printf("zip file seems valid")

	log.Printf("Starting http server...")
	l, err := net.Listen("tcp", "localhost:0")
	must(err)

	http.Handle("/", http.FileServer(&fakeFileSystem{zipBytes}))

	go func() {
		log.Fatal(http.Serve(l, nil))
	}()

	url := fmt.Sprintf("http://%s/file.dat", l.Addr().String())

	f, err := eos.Open(url, option.WithHTFSDumpStats(), option.WithHTFSCheck())
	must(err)
	defer f.Close()

	stats, err := f.Stat()
	fSize := stats.Size()

	numWorkers := 4
	indices := make(chan int)
	done := make(chan bool)

	var bytesExtracted int64
	var filesExtracted int64

	var printThreshold int64 = 50
	var running int64 = 1
	startTime := time.Now()

	work := func() {
		defer func() {
			done <- true
		}()

		zr, err := zip.NewReader(f, fSize)
		must(err)

		for index := range indices {
			func() {
				zf := zr.File[index]

				rc, err := zf.Open()
				must(err)

				copied, err := io.Copy(ioutil.Discard, rc)
				must(err)

				atomic.AddInt64(&bytesExtracted, copied)
				newFilesExtracted := atomic.AddInt64(&filesExtracted, 1)
				if (newFilesExtracted % printThreshold) == 0 {
					log.Printf("Extracted %d files (%d workers, running for %s)", newFilesExtracted, numWorkers, time.Since(startTime))
				}

				must(rc.Close())
			}()
		}
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		defer func() {
			close(indices)
		}()

		for {
			var indexSlice []int
			for i := 0; i < numFiles; i++ {
				indexSlice = append(indexSlice, i)
			}
			rand.Shuffle(len(indexSlice), func(i, j int) {
				indexSlice[i], indexSlice[j] = indexSlice[j], indexSlice[i]
			})

			for _, i := range indexSlice {
				if atomic.LoadInt64(&running) != 1 {
					log.Printf("Winding down...")
					return
				}

				indices <- i
			}
		}
	}()

	for i := 0; i < numWorkers; i++ {
		go work()
	}

	runningWorkers := numWorkers
	for runningWorkers > 0 {
		select {
		case <-done:
			runningWorkers--
		case <-sigChan:
			atomic.StoreInt64(&running, 0)
		}
	}

	log.Printf("Files extracted: %d", filesExtracted)
	log.Printf("Total extracted: %s", progress.FormatBytes(bytesExtracted))

	return nil
}
