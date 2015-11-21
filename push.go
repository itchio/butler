package main

import (
	"fmt"
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

func doPush(src string, repoSpec string) error {
	conn, err := wharf.Connect(*pushEndpoint, *pushIdentity)
	if err != nil {
		return err
	}
	defer conn.Close()

	uploadParams, err := bio.Marshal(bio.UploadParams{RepoSpec: repoSpec})
	if err != nil {
		return err
	}

	ok, _, err := conn.SendRequest("butler/upload-params", true, uploadParams)
	if err != nil {
		return err
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
		go func() {
			defer wg.Done()
			ch, err := conn.OpenCompressedChannel("butler/send-file", bio.FileAdded{Path: "/a/b/c"})
			if err != nil {
				errs <- err
				return
			}
			defer ch.Close()

			for i := 0; i < 8000; i++ {
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
