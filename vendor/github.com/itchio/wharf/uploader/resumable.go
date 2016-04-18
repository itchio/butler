package uploader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/dustin/go-humanize"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/splitfunc"
)

var seed = 0

// ResumableUpload keeps track of an upload and reports back on its progress
type ResumableUpload struct {
	c *itchio.Client

	TotalBytes    int64
	UploadedBytes int64
	OnProgress    func()

	// resumable URL as per GCS
	uploadURL string

	// where data is written so we can update counts
	writeCounter io.Writer

	// need to flush to squeeze all the data out
	bufferedWriter *bufio.Writer

	// need to close so reader end of pipe gets EOF
	pipeWriter io.Closer

	id       int
	consumer *pwr.StateConsumer
}

// Close flushes all intermediary buffers and closes the connection
func (ru *ResumableUpload) Close() error {
	var err error

	ru.Debugf("flushing buffered writer, %d written", ru.TotalBytes)

	err = ru.bufferedWriter.Flush()
	if err != nil {
		return err
	}

	ru.Debugf("closing pipe writer")

	err = ru.pipeWriter.Close()
	if err != nil {
		return err
	}

	ru.Debugf("closed pipe writer")
	ru.Debugf("everything closed! uploadedbytes = %d, totalbytes = %d", ru.UploadedBytes, ru.TotalBytes)

	return nil
}

// Write is our implementation of io.Writer
func (ru *ResumableUpload) Write(p []byte) (int, error) {
	return ru.writeCounter.Write(p)
}

func NewResumableUpload(uploadURL string, done chan bool, errs chan error, consumer *pwr.StateConsumer) (*ResumableUpload, error) {
	ru := &ResumableUpload{}
	ru.uploadURL = uploadURL
	ru.id = seed
	seed++
	ru.consumer = consumer
	ru.c = itchio.ClientWithKey("x")

	pipeR, pipeW := io.Pipe()

	ru.pipeWriter = pipeW

	// TODO: make configurable?
	const bufferSize = 32 * 1024 * 1024

	bufferedWriter := bufio.NewWriterSize(pipeW, bufferSize)
	ru.bufferedWriter = bufferedWriter

	onWrite := func(count int64) {
		ru.Debugf("onwrite %d", count)
		ru.TotalBytes = count
		if ru.OnProgress != nil {
			ru.OnProgress()
		}
	}
	ru.writeCounter = counter.NewWriterCallback(onWrite, bufferedWriter)

	go ru.uploadChunks(pipeR, done, errs)

	return ru, nil
}

func (ru *ResumableUpload) Debugf(f string, args ...interface{}) {
	ru.consumer.Debugf("[upload %d] %s", ru.id, fmt.Sprintf(f, args...))
}

const minBlockSize = 256 * 1024 // 256KB

func (ru *ResumableUpload) uploadChunks(reader io.Reader, done chan bool, errs chan error) {
	var offset int64 = 0

	sendBytes := func(buf []byte, isEnd bool) error {
		buflen := int64(len(buf))
		ru.Debugf("received %d bytes", buflen)

		body := bytes.NewReader(buf)
		countingReader := counter.NewReaderCallback(func(count int64) {
			ru.UploadedBytes = offset + count
			if ru.OnProgress != nil {
				ru.OnProgress()
			}
		}, body)

		req, err := http.NewRequest("PUT", ru.uploadURL, countingReader)
		if err != nil {
			return err
		}

		start := offset
		end := start + buflen - 1
		contentRange := fmt.Sprintf("bytes %d-%d/*", offset, end)

		if isEnd {
			contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, end, offset+buflen)
		}

		req.Header.Set("content-range", contentRange)

		res, err := ru.c.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 && res.StatusCode != 308 {
			ru.Debugf("uh oh, got HTTP %s", res.Status)
			resb, _ := ioutil.ReadAll(res.Body)
			ru.Debugf("server said %s", string(resb))
			return fmt.Errorf("HTTP %d while uploading", res.StatusCode)
		}

		offset += buflen
		ru.Debugf("%s uploaded, at %s", humanize.Bytes(uint64(offset)), res.Status)
		return nil
	}

	splitSize := 4 * minBlockSize

	s := bufio.NewScanner(reader)
	s.Buffer(make([]byte, splitSize), 0)
	s.Split(splitfunc.New(splitSize))

	buf1 := make([]byte, 0, splitSize)
	buf2 := make([]byte, 0, splitSize)

	for s.Scan() {
		buf2 = append(buf2[:0], buf1...)
		buf1 = append(buf1[:0], s.Bytes()...)

		if len(buf2) > 0 {
			ru.Debugf("sending %d block", len(buf2))
			err := sendBytes(buf2, false)
			if err != nil {
				errs <- err
				return
			}
		}
	}

	err := s.Err()
	if err != nil {
		ru.Debugf("scanner error :(")
		errs <- err
		return
	}

	ru.Debugf("sending last block, %d bytes", len(buf1))
	err = sendBytes(buf1, true)
	if err != nil {
		errs <- err
		return
	}

	done <- true
	ru.Debugf("done sent!")
}
