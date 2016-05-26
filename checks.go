package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/crc32c"
)

func checkIntegrity(resp *http.Response, totalBytes int64, file string) (bool, error) {
	diskSize := int64(0)
	stats, err := os.Lstat(file)
	if err == nil {
		diskSize = stats.Size()
	}

	if resp.ContentLength > 0 {
		if totalBytes != diskSize {
			return false, fmt.Errorf("Corrupt download: expected %d bytes, got %d", totalBytes, diskSize)
		}

		comm.Debugf("%10s pass (%d bytes)", "size", totalBytes)
	}

	return checkHashes(resp.Header, file)
}

func checkHashes(header http.Header, file string) (bool, error) {
	googHashes := header[http.CanonicalHeaderKey("x-goog-hash")]

	for _, googHash := range googHashes {
		tokens := strings.SplitN(googHash, "=", 2)
		hashType := tokens[0]
		hashValue, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			comm.Logf("Could not verify %s hash: %s", hashType, err)
			continue
		}

		start := time.Now()
		checked, err := checkHash(hashType, hashValue, file)
		if err != nil {
			return false, err
		}

		if checked {
			comm.Debugf("%10s pass (took %s)", hashType, time.Since(start))
		} else {
			comm.Debugf("%10s skip (use --thorough to force check)", hashType)
		}
	}

	return true, nil
}

func checkHash(hashType string, hashValue []byte, file string) (checked bool, err error) {
	checked = true

	switch hashType {
	case "md5":
		if *dlArgs.thorough {
			err = checkHashMD5(hashValue, file)
		} else {
			checked = false
		}
	case "crc32c":
		err = checkHashCRC32C(hashValue, file)
	default:
		checked = false
	}

	return
}

func checkHashMD5(hashValue []byte, file string) (err error) {
	fr, err := os.Open(file)
	if err != nil {
		return
	}
	defer fr.Close()

	hasher := md5.New()
	io.Copy(hasher, fr)

	hashComputed := hasher.Sum(nil)
	if !bytes.Equal(hashValue, hashComputed) {
		err = fmt.Errorf("md5 hash mismatch: got %x, expected %x", hashComputed, hashValue)
	}

	return
}

func checkHashCRC32C(hashValue []byte, file string) (err error) {
	fr, err := os.Open(file)
	if err != nil {
		return
	}
	defer fr.Close()

	hasher := crc32.New(crc32c.Table)
	io.Copy(hasher, fr)

	hashComputed := hasher.Sum(nil)
	if !bytes.Equal(hashValue, hashComputed) {
		err = fmt.Errorf("crc32c hash mismatch: got %x, expected %x", hashComputed, hashValue)
	}

	return
}
