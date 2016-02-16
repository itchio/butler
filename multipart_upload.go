package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/itchio/butler/comm"
)

type MultipartUpload struct {
	request        *http.Request
	bufferedWriter *bufio.Writer
	multiWriter    io.Closer
	pipeWriter     io.Closer
	partWriter     io.Writer
}

func (mu *MultipartUpload) Close() error {
	err := mu.multiWriter.Close()
	if err != nil {
		return err
	}

	err = mu.bufferedWriter.Flush()
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

	comm.Debugf("Creating multiwriter")
	const bufferSize = 16 * 1024 * 1024 // 16MB
	bufferedPipeW := bufio.NewWriterSize(pipeW, bufferSize)
	multiWriter := multipart.NewWriter(bufferedPipeW)

	comm.Debugf("Creating new HTTP request")
	req, err := http.NewRequest("POST", uploadURL, pipeR)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	go doReq(req, done, errs)

	for key, val := range uploadParams {
		comm.Debugf("Writing param %s=%s", key, val)
		err := multiWriter.WriteField(key, val)
		if err != nil {
			return nil, err
		}
	}

	comm.Debugf("Creating form file %s", fileName)
	partWriter, err := multiWriter.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	mu := &MultipartUpload{
		multiWriter:    multiWriter,
		bufferedWriter: bufferedPipeW,
		partWriter:     partWriter,
		pipeWriter:     pipeW,
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
		responseBytes, _ := ioutil.ReadAll(res.Body)
		errs <- fmt.Errorf("Server responded with HTTP %s to %s %s: %s", res.Status, res.Request.Method, res.Request.URL.String(), string(responseBytes))
	}

	done <- true
}
