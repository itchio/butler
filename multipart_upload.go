package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/itchio/butler/comm"
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

func newMultipartUpload(uploadURL string, uploadParams map[string]string, fileName string,
	done chan bool, errs chan error) (io.WriteCloser, error) {

	comm.Debugf("Creating pipe")
	pipeR, pipeW := io.Pipe()

	comm.Debugf("Creating new HTTP request")
	req, err := http.NewRequest("POST", uploadURL, pipeR)
	if err != nil {
		return nil, err
	}

	go doReq(req, done, errs)

	comm.Debugf("Creating multiwriter")
	multiWriter := multipart.NewWriter(pipeW)

	for key, val := range uploadParams {
		comm.Debugf("Writing param %s=%s", key, val)
		multiWriter.WriteField(key, val)
	}

	comm.Debugf("Creating form file %s", fileName)
	partWriter, err := multiWriter.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	mu := &MultipartUpload{
		multiWriter: multiWriter,
		partWriter:  partWriter,
		pipeWriter:  pipeW,
	}
	return mu, nil
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
