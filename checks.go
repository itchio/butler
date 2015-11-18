package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/itchio/butler/bcommon"
)

func checkIntegrity(resp *http.Response, totalBytes int64, file string) (bool, error) {
	diskSize := int64(0)
	stats, err := os.Lstat(file)
	if err == nil {
		diskSize = stats.Size()
	}

	if resp.ContentLength != 0 {
		bcommon.Msg(fmt.Sprintf("checking file size. should be %d, is %d", totalBytes, diskSize))

		if totalBytes != diskSize {
			return false, fmt.Errorf("corrupted downloaded: expected %d bytes, got %d", totalBytes, diskSize)
		}
	}

	return checkHashes(resp.Header, file)
}

func checkHashes(header http.Header, file string) (bool, error) {
	googHashes := header[http.CanonicalHeaderKey("x-goog-hash")]
	if len(googHashes) > 0 {
		bcommon.Msg(fmt.Sprintf("got %d goog-hashes to check", len(googHashes)))
	}

	for _, googHash := range googHashes {
		tokens := strings.SplitN(googHash, "=", 2)
		hashType := tokens[0]
		hashValue, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			bcommon.Msg(fmt.Sprintf("could not verify %s hash: %s", hashType, err))
			continue
		}

		start := time.Now()
		checked, err := checkHash(hashType, hashValue, file)
		if err != nil {
			return false, err
		}

		status := "pass"
		if !checked {
			status = "skipped"
		}
		bcommon.Msg(fmt.Sprintf("%s hash: %s (in %s)", hashType, status, time.Since(start)))
	}

	return true, nil
}

func checkHash(hashType string, hashValue []byte, file string) (bool, error) {
	switch hashType {
	case "md5":
		return checkHashMD5(hashValue, file)
	}

	return false, nil
}

func checkHashMD5(hashValue []byte, file string) (bool, error) {
	fr, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer fr.Close()

	hasher := md5.New()
	io.Copy(hasher, fr)

	hashComputed := hasher.Sum(nil)
	if !bytes.Equal(hashValue, hashComputed) {
		return false, fmt.Errorf("md5 hash mismatch: got %x, expected %x", hashComputed, hashValue)
	}

	return true, nil
}
