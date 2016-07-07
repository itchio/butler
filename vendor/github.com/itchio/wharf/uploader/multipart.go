package uploader

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/go-errors/errors"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
)

// MultipartUpload keeps track of an upload and reports back on its progress
type MultipartUpload struct {
	TotalBytes    int64
	UploadedBytes int64
	OnProgress    func()

	// underlying writer
	partWriter io.Writer

	// owns partWriter, need to close to write boundary
	multiWriter io.Closer

	// need to flush to squeeze all the data out
	bufferedWriter *bufio.Writer

	// need to close so reader end of pipe gets EOF
	pipeWriter io.Closer
}

// Close flushes all intermediary buffers and closes the connection
func (mu *MultipartUpload) Close() error {
	err := mu.multiWriter.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = mu.bufferedWriter.Flush()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = mu.pipeWriter.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

// Write is our implementation of io.Writer
func (mu *MultipartUpload) Write(p []byte) (int, error) {
	return mu.partWriter.Write(p)
}

func NewMultipartUpload(uploadURL string, uploadParams map[string]string, fileName string,
	done chan bool, errs chan error, consumer *pwr.StateConsumer) (*MultipartUpload, error) {

	mu := &MultipartUpload{}

	pipeR, pipeW := io.Pipe()

	mu.pipeWriter = pipeW

	// TODO: make configurable?
	const bufferSize = 32 * 1024 * 1024

	bufferedWriter := bufio.NewWriterSize(pipeW, bufferSize)
	mu.bufferedWriter = bufferedWriter

	onWrite := func(count int64) {
		mu.TotalBytes = count
		if mu.OnProgress != nil {
			mu.OnProgress()
		}
	}
	writeCounter := counter.NewWriterCallback(onWrite, bufferedWriter)

	multiWriter := multipart.NewWriter(writeCounter)
	mu.multiWriter = multiWriter

	onRead := func(count int64) {
		mu.UploadedBytes = count
		if mu.OnProgress != nil {
			mu.OnProgress()
		}
	}
	readCounter := counter.NewReaderCallback(onRead, pipeR)

	consumer.Debug("Creating new HTTP request")
	req, err := http.NewRequest("POST", uploadURL, readCounter)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	req.Header.Set("Content-Type", multiWriter.FormDataContentType())

	go doReq(req, done, errs)

	for key, val := range uploadParams {
		consumer.Debugf("Writing param %s=%s", key, val)
		err := multiWriter.WriteField(key, val)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	consumer.Debugf("Creating form file %s", fileName)
	partWriter, err := multiWriter.CreateFormFile("file", fileName)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	mu.partWriter = partWriter

	return mu, nil
}

func doReq(req *http.Request, done chan bool, errs chan error) {
	c := itchio.ClientWithKey("x")

	res, err := c.Do(req)
	if err != nil {
		errs <- errors.Wrap(err, 1)
		return
	}

	if res.StatusCode/100 != 2 {
		responseBytes, _ := ioutil.ReadAll(res.Body)
		errs <- errors.Wrap(fmt.Errorf("Server responded with HTTP %s to %s %s: %s", res.Status, res.Request.Method, res.Request.URL.String(), string(responseBytes)), 1)
	}

	done <- true
}
