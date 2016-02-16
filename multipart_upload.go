package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type MultipartUpload struct {
	request     *http.Request
	multiWriter io.Closer
	pipeWriter  io.Closer
	partWriter  io.Writer
}

func (mu *MultipartUpload) Close() error {
	err := mu.multiWriter.Close()
	if err != nil {
		return err
	}

	err = mu.pipeWriter.Close()
	if err != nil {
		return err
	}

	return nil
}

func (mu *MultipartUpload) Write(p []byte) (int, error) {
	return mu.partWriter.Write(p)
}

func newMultipartUpload(uploadURL string, fileName string) (*http.Request, io.WriteCloser, error) {
	pipeR, pipeW := io.Pipe()

	multiWriter := multipart.NewWriter(pipeW)
	partWriter, err := multiWriter.CreateFormFile("file", fileName)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest("POST", uploadURL, pipeR)
	if err != nil {
		return nil, nil, err
	}

	req.ContentLength = -1

	mu := &MultipartUpload{
		multiWriter: multiWriter,
		partWriter:  partWriter,
		pipeWriter:  pipeW,
	}
	return req, mu, nil
}

func doReq(req *http.Request, done chan bool, errs chan error) {
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		errs <- err
	}

	if res.StatusCode/100 != 2 {
		errs <- fmt.Errorf("Server responded with HTTP %d %s", res.StatusCode, res.Status)
	}

	done <- true
}
