package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/itchio/butler/bcommon"
)

const bufferSize = 128 * 1024

func dl() {
	if len(os.Args) < 4 {
		bcommon.Die("Missing url or dest for dl command")
	}
	url := os.Args[2]
	dest := os.Args[3]

	tries := 3
	for tries > 0 {
		_, err := tryDl(url, dest)
		if err == nil {
			break
		}

		bcommon.Msg(fmt.Sprintf("While downloading, got error %s", err))
		tries--
		if tries > 0 {
			os.Truncate(dest, 0)
			bcommon.Msg(fmt.Sprintf("Retrying... (%d tries left)", tries))
		} else {
			bcommon.Die(err.Error())
		}
	}
}

func tryDl(url string, dest string) (int64, error) {
	existingBytes := int64(0)
	stats, err := os.Lstat(dest)
	if err == nil {
		existingBytes = stats.Size()
	}

	bcommon.Msg(fmt.Sprintf("existing file is %d bytes long", existingBytes))

	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)
	byteRange := fmt.Sprintf("bytes=%d-", existingBytes)
	bcommon.Msg(fmt.Sprintf("Asking for range %s", byteRange))

	req.Header.Set("Range", byteRange)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	doDownload := true
	totalBytes := existingBytes + resp.ContentLength

	switch resp.StatusCode {
	case 200: // OK
		// will send data, but doesn't support byte ranges
		existingBytes = 0
		totalBytes = resp.ContentLength
		os.Truncate(dest, 0)
	case 206: // Partial Content
		// will send incremental data
	case 416: // Requested Range not Satisfiable
		// already has everything
		doDownload = false

		req, _ := http.NewRequest("HEAD", url, nil)
		resp, err = client.Do(req)
		if err != nil {
			return 0, err
		}

		if existingBytes > resp.ContentLength {
			bcommon.Msg(fmt.Sprintf("existing file too big (%d), truncating to %d", existingBytes, resp.ContentLength))
			existingBytes = resp.ContentLength
			os.Truncate(dest, existingBytes)
		}
		totalBytes = existingBytes
	default:
		return 0, fmt.Errorf("server error: http %s", resp.Status)
	}

	if doDownload {
		bcommon.Msg(fmt.Sprintf("Response content length = %d", resp.ContentLength))
		_, err := appendAllToFile(resp.Body, dest, existingBytes, totalBytes)
		if err != nil {
			return 0, err
		}
		bcommon.Msg(fmt.Sprintf("done downloading"))
	} else {
		bcommon.Msg(fmt.Sprintf("all downloaded already"))
	}

	_, err = checkIntegrity(resp, totalBytes, dest)
	if err != nil {
		return 0, err
	}

	return totalBytes, nil
}

func appendAllToFile(src io.Reader, dest string, existingBytes int64, totalBytes int64) (int64, error) {
	out, _ := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer out.Close()

	bytesWritten := existingBytes
	for {
		n, err := io.CopyN(out, src, bufferSize)
		bytesWritten += n

		percent := int(bytesWritten * 100 / totalBytes)
		status := &bcommon.ButlerDownloadStatus{Percent: percent}
		bcommon.Send(status)

		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	return bytesWritten, nil
}
