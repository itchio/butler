package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/itchio/rsync-go.v0"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/bio"
	"github.com/itchio/butler/wharf"
)

func push(src string, repoSpec string) {
	err := doPush(src, repoSpec)
	if err != nil {
		bio.Die(err.Error())
	}
}

var (
	addressRe   = regexp.MustCompile("[^:]:.*")
	defaultPort = 22
)

func doPush(src string, repoSpec string) error {
	address := *pushAddress
	if !addressRe.MatchString(address) {
		address = fmt.Sprintf("%s:%d", address, defaultPort)
	}

	conn, err := wharf.Connect(address, *pushIdentity, version)
	if err != nil {
		return err
	}
	defer conn.Close()

	up := bio.UploadParams{RepoSpec: repoSpec}
	ok, _, err := conn.SendRequest("butler/upload-params", true, up)
	if err != nil {
		return fmt.Errorf("Could not send upload params: %s", err.Error())
	}

	if !ok {
		return fmt.Errorf("Could not find upload to replace from '%s'", repoSpec)
	}
	bio.Log("upload params were accepted!")

	var wg sync.WaitGroup

	done := make(chan bool)
	errs := make(chan error)

	log.Printf("\n")
	var totalSize int64
	fileList := make(map[string]os.FileInfo)

	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errs <- err
			return err
		}

		if info.IsDir() && len(path) > 1 && strings.HasPrefix(info.Name(), ".") {
			log.Printf("Skipping %s because it starts with .\n", path)
			return filepath.SkipDir
		}

		if !info.IsDir() {
			fileList[path] = info
		}
		return nil
	})

	for path, info := range fileList {
		totalSize += info.Size()
		humanSize := humanize.Bytes(uint64(info.Size()))
		log.Printf("%8s | %s\n", humanSize, path)
	}
	log.Printf("---------+-------------\n")
	log.Printf("%8s | (total)\n", humanize.Bytes(uint64(totalSize)))

	for j := 0; j < 10; j++ {
		wg.Add(1)
		j := j
		go func() {
			defer wg.Done()
			path := fmt.Sprintf("/test/channel/%d", j)
			ch, err := conn.OpenCompressedChannel("butler/send-file", bio.FileAdded{Path: path})
			if err != nil {
				errs <- err
				return
			}
			defer ch.Close()

			for i := 0; i < 250; i++ {
				err := ch.Send(bio.SourceFile{
					Path:   fmt.Sprintf(""),
					Hashes: []rsync.BlockHash{},
				})
				if err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case err := <-errs:
		return err
	case <-done:
		return nil
	}
}
