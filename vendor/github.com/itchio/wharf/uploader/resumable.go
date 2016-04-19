package uploader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/timeout"
)

var seed = 0

func fromEnv(envName string, defaultValue int) int {
	v := os.Getenv(envName)
	if v != "" {
		iv, err := strconv.Atoi(v)
		if err == nil {
			log.Printf("Override set: %s = %d", envName, iv)
			return iv
		}
	}
	return defaultValue
}

var resumableMaxRetries = fromEnv("WHARF_MAX_RETRIES", 20)
var resumableConnectTimeout = time.Duration(fromEnv("WHARF_CONNECT_TIMEOUT", 10)) * time.Second
var resumableIdleTimeout = time.Duration(fromEnv("WHARF_IDLE_TIMEOUT", 15)) * time.Second

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
	ru.c.HTTPClient = timeout.NewClient(resumableConnectTimeout, resumableIdleTimeout)

	pipeR, pipeW := io.Pipe()

	ru.pipeWriter = pipeW

	// TODO: make configurable?
	const bufferSize = 32 * 1024 * 1024

	bufferedWriter := bufio.NewWriterSize(pipeW, bufferSize)
	ru.bufferedWriter = bufferedWriter

	onWrite := func(count int64) {
		// ru.Debugf("onwrite %d", count)
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

const minChunkSize = 256 * 1024 // 256KB
const maxChunkGroup = 64
const maxSendBuf = maxChunkGroup * minChunkSize // 16MB

type blockItem struct {
	buf    []byte
	isLast bool
}

type netError struct {
	err error
}

func (ne *netError) Error() string {
	return fmt.Sprintf("network error: %s", ne.err.Error())
}

func (ru *ResumableUpload) uploadChunks(reader io.Reader, done chan bool, errs chan error) {
	var offset int64 = 0

	sendBuf := make([]byte, 0, maxSendBuf)
	reqBlocks := make(chan blockItem, maxChunkGroup)

	canceller := make(chan bool)

	doSendBytesOnce := func(buf []byte, isLast bool) error {
		buflen := int64(len(sendBuf))
		ru.Debugf("uploading chunk of %d bytes", buflen)

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
		ru.Debugf("uploading %d-%d", start, end)

		if isLast {
			contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, end, offset+buflen)
		}

		req.Header.Set("content-range", contentRange)

		res, err := ru.c.Do(req)
		if err != nil {
			ru.Debugf("while uploading %d-%d: \n%s", start, end, err.Error())
			return &netError{err}
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

	doSendBytes := func(buf []byte, isLast bool) error {
		tries := 1

		for tries < resumableMaxRetries {
			err := doSendBytesOnce(buf, isLast)
			if err != nil {
				if ne, ok := err.(*netError); ok {
					delay := tries * tries
					ru.consumer.PauseProgress()
					ru.consumer.Infof("")
					ru.consumer.Infof("%s", ne.Error())
					ru.consumer.Infof("Sleeping %d seconds then retrying", delay)
					time.Sleep(time.Second * time.Duration(delay))
					ru.consumer.ResumeProgress()
					tries++
					continue
				} else {
					return err
				}
			} else {
				return nil
			}
		}

		return fmt.Errorf("Too many network errors, giving up.")
	}

	s := bufio.NewScanner(reader)
	s.Buffer(make([]byte, minChunkSize), 0)
	s.Split(splitfunc.New(minChunkSize))

	// we need two buffers to know when we're at EOF,
	// for sizes that are an exact multiple of minChunkSize
	buf1 := make([]byte, 0, minChunkSize)
	buf2 := make([]byte, 0, minChunkSize)

	subDone := make(chan bool)
	subErrs := make(chan error)

	ru.Debugf("kicking off sender")

	go func() {
		isLast := false

		for !isLast {
			sendBuf = sendBuf[:0]

			for len(sendBuf) < maxSendBuf && !isLast {
				var item blockItem
				if len(sendBuf) == 0 {
					ru.Debugf("sender blocking receive")
					select {
					case item = <-reqBlocks:
						// cool
					case <-canceller:
						ru.Debugf("send cancelled")
						return
					}
				} else {
					ru.Debugf("sender non-blocking receive")
					select {
					case item = <-reqBlocks:
						// cool
					case <-canceller:
						ru.Debugf("send cancelled")
						return
					default:
						ru.Debugf("sent faster than scanned, uploading smaller chunk")
						break
					}
				}

				if item.isLast {
					isLast = true
				}

				sendBuf = append(sendBuf, item.buf...)
			}

			if len(sendBuf) > 0 {
				err := doSendBytes(sendBuf, isLast)
				if err != nil {
					ru.Debugf("send error, bailing out")
					subErrs <- err
					return
				}
			}
		}

		subDone <- true
		ru.Debugf("sender done")
	}()

	scannedBufs := make(chan []byte)
	usedBufs := make(chan bool)

	go func() {
		for s.Scan() {
			select {
			case scannedBufs <- s.Bytes():
				// woo
			case <-canceller:
				ru.Debugf("scan cancelled (1)")
				break
			}
			select {
			case <-usedBufs:
				// woo
			case <-canceller:
				ru.Debugf("scan cancelled (2)")
				break
			}
		}
		close(scannedBufs)
	}()

	// break patch into chunks of minChunkSize, signal last block
	go func() {
		for scannedBuf := range scannedBufs {
			buf2 = append(buf2[:0], buf1...)
			buf1 = append(buf1[:0], scannedBuf...)
			usedBufs <- true

			// all but first iteration
			if len(buf2) > 0 {
				select {
				case reqBlocks <- blockItem{buf: append([]byte{}, buf2...), isLast: false}:
					// okay cool let's go c'mon
				case <-canceller:
					ru.Debugf("scan cancelled (3)")
					return
				}
			}
		}

		err := s.Err()
		if err != nil {
			ru.Debugf("scanner error :(")
			subErrs <- err
			return
		}

		select {
		case reqBlocks <- blockItem{buf: append([]byte{}, buf1...), isLast: true}:
		case <-canceller:
			ru.Debugf("scan cancelled (right near the finish line)")
			return
		}

		subDone <- true
		ru.Debugf("scanner done")
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-subDone:
			// woo!
		case err := <-subErrs:
			ru.Debugf("got sub error: %s, bailing", err.Error())
			close(canceller)
			errs <- err
			return
		}
	}

	done <- true
	ru.Debugf("done sent!")
}
