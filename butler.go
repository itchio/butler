package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

var version = "head"

type butlerError struct {
	Error string
}

type butlerDownloadStatus struct {
	Percent int
}

const bufferSize = 128 * 1024

func main() {
	if len(os.Args) < 2 {
		err("Missing command")
	}
	cmd := os.Args[1]

	switch cmd {
	case "version":
		fmt.Println(fmt.Sprintf("butler version %s", version))
	case "dl":
		dl()
	default:
		err("Invalid command")
	}
}

func send(v interface{}) {
	j, _ := json.Marshal(v)
	fmt.Println(string(j))
}

func err(msg string) {
	e := &butlerError{Error: msg}
	send(e)
	os.Exit(1)
}

func dl() {
	if len(os.Args) < 4 {
		err("Missing url or dest for dl command")
	}
	url := os.Args[2]
	dest := os.Args[3]

	initialBytes := int64(0)
	stats, err := os.Lstat(dest)
	if err == nil {
		initialBytes = stats.Size()
	}

	bytesWritten := initialBytes

	out, _ := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	defer out.Close()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", bytesWritten))
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	for {
		n, _ := io.CopyN(out, resp.Body, bufferSize)
		bytesWritten += n

		totalBytes := (initialBytes + resp.ContentLength)
		status := &butlerDownloadStatus{
			Percent: int(bytesWritten * 100 / totalBytes)}
		send(status)

		if n == 0 {
			break
		}
	}
}
