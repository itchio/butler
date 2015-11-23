package main

import (
	"fmt"
	"regexp"
	"sync"

	"gopkg.in/itchio/rsync-go.v0"

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
		return fmt.Errorf("could not send upload params: %s", err.Error())
	}

	if !ok {
		return fmt.Errorf("could not find upload to replace from repo spec '%s' - be more specific ?", repoSpec)
	}
	bio.Log("upload params were accepted!")

	var wg sync.WaitGroup

	done := make(chan bool)
	errs := make(chan error)

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
