package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func split(target string, manifest string) {
	must(doSplit(target, manifest))
}

func doSplit(target string, manifest string) error {
	bigBlockSize := int(*appArgs.bigBlockSize)

	stats, err := os.Lstat(target)
	if err != nil {
		return err
	}

	file, err := os.Open(target)
	if err != nil {
		return err
	}

	zr, err := zip.NewReader(file, stats.Size())
	if err != nil {
		return err
	}

	container, err := tlc.WalkZip(zr, filterPaths)
	if err != nil {
		return err
	}

	comm.Statf("splitting %s in %s", humanize.IBytes(uint64(container.Size)), container.Stats())

	startTime := time.Now()

	done := make(chan bool)
	errs := make(chan error)

	blockDir := "blocks"

	hashes := make([][][]byte, len(container.Files))
	for i, f := range container.Files {
		numBlocks := int(math.Ceil(float64(f.Size) / float64(bigBlockSize)))
		hashes[i] = make([][]byte, numBlocks)
	}

	for i := range container.Files {
		go func(fileIndex int64) {
			pool := container.NewZipPool(zr)
			r, err := pool.GetReader(int64(fileIndex))
			if err != nil {
				errs <- err
				return
			}

			shake128 := sha3.NewShake128()
			hashBuf := make([]byte, 32)

			s := bufio.NewScanner(r)
			s.Buffer(make([]byte, bigBlockSize), 0)
			s.Split(splitfunc.New(bigBlockSize))

			blockIndex := 0

			for s.Scan() {
				buf := s.Bytes()

				shake128.Reset()
				_, err = shake128.Write(buf)
				if err != nil {
					errs <- err
					return
				}

				_, err = io.ReadFull(shake128, hashBuf)
				if err != nil {
					errs <- err
					return
				}

				comm.Logf("len(hashes) = %d", len(hashes))
				comm.Logf("fileIndex = %d", fileIndex)
				comm.Logf("len(hashes[fileIndex]) = %d", len(hashes[fileIndex]))
				hashes[fileIndex][blockIndex] = append([]byte{}, hashBuf...)

				blockPath := filepath.Join(blockDir, "shake128-32", fmt.Sprintf("%x", hashBuf), fmt.Sprintf("%d", len(buf)))
				comm.Logf("%s", blockPath)

				err = os.MkdirAll(filepath.Dir(blockPath), 0755)
				if err != nil {
					errs <- err
					return
				}

				var f *os.File
				f, err = os.Create(blockPath)
				if err != nil {
					errs <- err
					return
				}

				_, err = f.Write(buf)
				if err != nil {
					errs <- err
					return
				}

				err = f.Close()
				if err != nil {
					errs <- err
					return
				}

				blockIndex++
			}

			err = pool.Close()
			if err != nil {
				errs <- err
				return
			}

			done <- true
		}(int64(i))
	}

	for i := 0; i < len(container.Files); i++ {
		select {
		case err := <-errs:
			panic(err)
		case <-done:
			// all good
		}
	}

	duration := time.Since(startTime)
	perSec := humanize.IBytes(uint64(float64(container.Size) / duration.Seconds()))

	comm.Statf("Read everything in %s (%s/s)", duration, perSec)

	// write manifest
	manifestWriter, err := os.Create(*splitArgs.manifest)
	if err != nil {
		return err
	}

	rawManWire := wire.NewWriteContext(manifestWriter)
	err = rawManWire.WriteMagic(pwr.ManifestMagic)
	if err != nil {
		return err
	}

	compression := butlerCompressionSettings()
	err = rawManWire.WriteMessage(&pwr.ManifestHeader{
		Compression: &compression,
		Algorithm:   pwr.HashAlgorithm_SHAKE128_32,
	})
	if err != nil {
		return err
	}

	manWire, err := pwr.CompressWire(rawManWire, &compression)
	if err != nil {
		return err
	}

	err = manWire.WriteMessage(container)
	if err != nil {
		return err
	}

	sh := &pwr.SyncHeader{}
	mbh := &pwr.ManifestBlockHash{}

	for i := range container.Files {
		sh.Reset()
		sh.FileIndex = int64(i)
		err = manWire.WriteMessage(sh)
		if err != nil {
			return err
		}

		for _, hash := range hashes[i] {
			mbh.Reset()
			mbh.Hash = hash
			err = manWire.WriteMessage(mbh)
			if err != nil {
				return err
			}
		}
	}

	err = manWire.Close()
	if err != nil {
		return err
	}

	return nil
}
