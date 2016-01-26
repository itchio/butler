package main

import (
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/getlantern/idletiming"
	"github.com/itchio/wharf/counter"
)

const bufferSize = 128 * 1024

func dl(url string, dest string) {
	_, err := tryDl(url, dest)
	if err != nil {
		Die(err.Error())
	}
}

func timeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		idleConn := idletiming.Conn(conn, rwTimeout, func() {
			Logf("connection was idle for too long, dropping")
			conn.Close()
		})
		return idleConn, nil
	}
}

func newTimeoutClient(connectTimeout time.Duration, readWriteTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: timeoutDialer(connectTimeout, readWriteTimeout),
		},
	}
}

func tryDl(url string, dest string) (int64, error) {
	existingBytes := int64(0)
	stats, err := os.Lstat(dest)
	if err == nil {
		existingBytes = stats.Size()
	}

	client := newTimeoutClient(30*time.Second, 60*time.Second)

	req, _ := http.NewRequest("GET", url, nil)
	byteRange := fmt.Sprintf("bytes=%d-", existingBytes)

	req.Header.Set("Range", byteRange)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doDownload := true
	totalBytes := existingBytes + resp.ContentLength

	hostInfo := fmt.Sprintf("%s at %s", resp.Header.Get("Server"), req.Host)

	switch resp.StatusCode {
	case 200: // OK
		if *appArgs.verbose {
			Logf("HTTP 200 OK (no byte range support)")
		}
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
		if *appArgs.verbose {
			Logf("HTTP 206 Partial Content")
		}
		// will send incremental data
	case 416: // Requested Range not Satisfiable
		if *appArgs.verbose {
			Logf("HTTP 416 Requested Range not Satisfiable")
		}
		// already has everything
		doDownload = false

		req, _ := http.NewRequest("HEAD", url, nil)
		resp, err = client.Do(req)
		if err != nil {
			return 0, err
		}

		if existingBytes > resp.ContentLength {
			if *appArgs.verbose {
				Logf("Existing file too big (%d), truncating to %d", existingBytes, resp.ContentLength)
			}
			existingBytes = resp.ContentLength
			os.Truncate(dest, existingBytes)
		}
		totalBytes = existingBytes
	default:
		return 0, fmt.Errorf("%s responded with HTTP %s", hostInfo, resp.Status)
	}

	if doDownload {
		if existingBytes > 0 {
			Logf("Resuming (%s + %s = %s) download from %s", humanize.Bytes(uint64(existingBytes)), humanize.Bytes(uint64(resp.ContentLength)), humanize.Bytes(uint64(totalBytes)), hostInfo)
		} else {
			Logf("Downloading %s from %s", humanize.Bytes(uint64(resp.ContentLength)), hostInfo)
		}
		err := appendAllToFile(resp.Body, dest, existingBytes, totalBytes)
		if err != nil {
			return 0, err
		}
	} else {
		Log("Already fully downloaded")
	}

	_, err = checkIntegrity(resp, totalBytes, dest)
	if err != nil {
		Log("integrity checks failed, truncating")
		os.Truncate(dest, 0)
		return 0, err
	}

	return totalBytes, nil
}

func appendAllToFile(src io.Reader, dest string, existingBytes int64, totalBytes int64) (err error) {
	out, _ := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer out.Close()

	prevPercent := 0.0

	onWrite := func(bytesDownloaded int64) {
		bytesWritten := existingBytes + bytesDownloaded
		percent := float64(bytesWritten) * 100.0 / float64(totalBytes)
		if math.Abs(percent-prevPercent) < 0.1 {
			return
		}

		prevPercent = percent
		Progress(percent)
	}
	counter := counter.NewWriterCallback(onWrite, out)

	_, err = io.Copy(counter, src)
	EndProgress()
	return
}
