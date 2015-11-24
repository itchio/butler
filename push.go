package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/itchio/rsync-go.v0"
	"gopkg.in/kothar/brotli-go.v0/enc"

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

	var wg sync.WaitGroup

	var totalBops int64

	wg.Add(1)
	go func() {
		defer wg.Done()
		for req := range conn.Reqs {
			payload, err := wharf.GetPayload(req)
			if err != nil {
				log.Printf("Error receiving payload: %s\n", err.Error())
				conn.Close()
			}

			switch v := payload.(type) {
			case bio.LogEntry:
				log.Printf("remote: %s\n", v.Message)
			case bio.SourceFile:
				var localSize int64 = 0
				path := path.Join(src, v.Path)
				stats, err := os.Lstat(path)
				if err == nil {
					localSize = stats.Size()
				}

				bdiff := int64(v.Size) - localSize

				out := new(bytes.Buffer)
				params := enc.NewBrotliParams()
				params.SetQuality(1)
				bw := enc.NewBrotliWriter(params, out)

				genc := gob.NewEncoder(bw)
				nops := 0

				opWriter := func(op rsync.Operation) error {
					nops++
					genc.Encode(op)
					return nil
				}
				rs := &rsync.RSync{}
				fr, err := os.Open(path)
				if err != nil {
					log.Printf("Error opening %s: %s\n", path, err.Error())
					return
				}
				defer fr.Close()
				err = rs.CreateDelta(fr, v.Hashes, opWriter, nil)
				if err != nil {
					log.Printf("Error creating delta for %s: %s\n", path, err.Error())
					return
				}

				bw.Close()
				log.Printf("%30s | %d bytes, %d ops taking %s", path, bdiff, nops, humanize.Bytes(uint64(out.Len())))
				totalBops += int64(out.Len())
			default:
				log.Printf("Server sent us unknown req %s\n", req.Type)
			}
		}
		log.Println("Done handing reqs from server")
		log.Printf("total bops = %d bytes, %s\n", totalBops, humanize.Bytes(uint64(totalBops)))
	}()

	up := bio.UploadParams{RepoSpec: repoSpec}
	ok, _, err := conn.SendRequest("butler/upload-params", true, up)
	if err != nil {
		return fmt.Errorf("Could not send upload params: %s", err.Error())
	}

	if !ok {
		return fmt.Errorf("Could not find upload to replace from '%s'", repoSpec)
	}
	bio.Log("upload params were accepted!")

	done := make(chan bool)
	errs := make(chan error)

	log.Printf("\n")
	var totalSize uint64
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
		fileSize := uint64(info.Size())
		totalSize += fileSize
		humanSize := humanize.Bytes(fileSize)
		log.Printf("%8s | %s\n", humanSize, path)
	}
	log.Printf("---------+-------------\n")
	log.Printf("%8s | (total)\n", humanize.Bytes(totalSize))

	for j := 0; j < 0; j++ {
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
					Path:   "",
					Hashes: []rsync.BlockHash{},
				})
				if err != nil {
					errs <- err
					return
				}
			}
			log.Printf("Done sending through channel %s\n", path)
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
