package main

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/counter"
)

const bufferSize = 128 * 1024

func dl(url string, dest string) {
	_, err := tryDl(url, dest)
	if err != nil {
		comm.Die(err.Error())
	}
}

func tryDl(url string, dest string) (int64, error) {
	existingBytes := int64(0)
	stats, err := os.Lstat(dest)
	if err == nil {
		existingBytes = stats.Size()
	}

	client := timeout.NewDefaultClient()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("User-Agent", userAgent())
	byteRange := fmt.Sprintf("bytes=%d-", existingBytes)

	req.Header.Set("Range", byteRange)
	resp, err := client.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, 1)
	}
	defer resp.Body.Close()

	doDownload := true
	totalBytes := existingBytes + resp.ContentLength

	hostInfo := fmt.Sprintf("%s at %s", resp.Header.Get("Server"), req.Host)

	switch resp.StatusCode {
	case 200: // OK
		comm.Debugf("HTTP 200 OK (no byte range support)")
		totalBytes = resp.ContentLength

		if existingBytes == resp.ContentLength {
			// already have the exact same number of bytes, hopefully the same ones
			doDownload = false
		} else {
			// will send data, but doesn't support byte ranges
			existingBytes = 0
			os.Truncate(dest, 0)
		}
	case 206: // Partial Content
		comm.Debugf("HTTP 206 Partial Content")
		// will send incremental data
	case 416: // Requested Range not Satisfiable
		comm.Debugf("HTTP 416 Requested Range not Satisfiable")
		// already has everything
		doDownload = false

		// the request we just made failed, so let's make another one
		// and close it immediately. this will get us the right content
		// length and any checksums the server might have to offer
		// Note: we'd use HEAD here but a bunch of servers don't
		// reply with a proper content-length.

		// closing the old one first...
		resp.Body.Close()

		req, _ = http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", userAgent())

		resp, err = client.Do(req)
		if err != nil {
			return 0, errors.Wrap(err, 1)
		}
		// immediately close new request, we're only interested
		// in headers.
		resp.Body.Close()

		if existingBytes > resp.ContentLength {
			comm.Debugf("Existing file too big (%d), truncating to %d", existingBytes, resp.ContentLength)
			existingBytes = resp.ContentLength
			os.Truncate(dest, existingBytes)
		}
		totalBytes = existingBytes
	default:
		return 0, fmt.Errorf("%s responded with HTTP %s", hostInfo, resp.Status)
	}

	if doDownload {
		if existingBytes > 0 {
			comm.Logf("Resuming (%s + %s = %s) download from %s", humanize.IBytes(uint64(existingBytes)), humanize.IBytes(uint64(resp.ContentLength)), humanize.IBytes(uint64(totalBytes)), hostInfo)
		} else {
			comm.Logf("Downloading %s from %s", humanize.IBytes(uint64(resp.ContentLength)), hostInfo)
		}
		err = appendAllToFile(resp.Body, dest, existingBytes, totalBytes)
		if err != nil {
			return 0, errors.Wrap(err, 1)
		}
	} else {
		comm.Log("Already fully downloaded")
	}

	err = checkIntegrity(resp.Header, totalBytes, dest)
	if err != nil {
		comm.Log("Integrity checks failed, truncating")
		os.Truncate(dest, 0)
		return 0, errors.Wrap(err, 1)
	}

	return totalBytes, nil
}

func appendAllToFile(src io.Reader, dest string, existingBytes int64, totalBytes int64) error {
	out, _ := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer out.Close()

	prevPercent := 0.0
	comm.StartProgress()

	onWrite := func(bytesDownloaded int64) {
		bytesWritten := existingBytes + bytesDownloaded
		percent := float64(bytesWritten) / float64(totalBytes)
		if math.Abs(percent-prevPercent) < 0.0001 {
			return
		}

		prevPercent = percent
		comm.Progress(percent)
	}
	counter := counter.NewWriterCallback(onWrite, out)

	_, err := io.Copy(counter, src)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	comm.EndProgress()
	return nil
}
