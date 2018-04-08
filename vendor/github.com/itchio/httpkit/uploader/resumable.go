package uploader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/splitfunc"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

var seed = 0

var resumableMaxRetries = fromEnv("WHARF_MAX_RETRIES", 15)
var resumableConnectTimeout = time.Duration(fromEnv("WHARF_CONNECT_TIMEOUT", 30)) * time.Second
var resumableIdleTimeout = time.Duration(fromEnv("WHARF_IDLE_TIMEOUT", 60)) * time.Second
var resumableVerboseDebug = fromEnv("WHARF_VERBOSE_DEBUG", 0)

// ResumableUpload keeps track of an upload and reports back on its progress
type ResumableUpload struct {
	httpClient *http.Client

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
	consumer *state.Consumer

	MaxChunkGroup int
	BufferSize    int
}

type ResumableUploadSettings struct {
	/** Consumer gets progress info, debug messages, etc. */
	Consumer *state.Consumer

	/* MaxChunkGroup is how many 256KB chunks we'll try to upload in a single request to GCS. Defaults to 64 (16MiB) */
	MaxChunkGroup int

	/* BufferSize is how much data we'll accept from the writer before blocking. Defaults to 32MiB. */
	BufferSize int
}

func NewResumableUpload(uploadURL string, done chan bool, errs chan error, settings ResumableUploadSettings) (*ResumableUpload, error) {
	ru := &ResumableUpload{}
	ru.MaxChunkGroup = settings.MaxChunkGroup
	if ru.MaxChunkGroup == 0 {
		ru.MaxChunkGroup = 64
	}
	ru.uploadURL = uploadURL
	ru.id = seed
	seed++
	ru.consumer = settings.Consumer
	ru.httpClient = timeout.NewClient(resumableConnectTimeout, resumableIdleTimeout)

	pipeR, pipeW := io.Pipe()

	ru.pipeWriter = pipeW

	bufferSize := settings.BufferSize
	if bufferSize == 0 {
		bufferSize = 32 * 1024 * 1024
	}

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

// Close flushes all intermediary buffers and closes the connection
func (ru *ResumableUpload) Close() error {
	var err error

	ru.Debugf("flushing buffered writer, %d written", ru.TotalBytes)

	err = ru.bufferedWriter.Flush()
	if err != nil {
		return errors.WithStack(err)
	}

	ru.Debugf("closing write end of pipe")

	err = ru.pipeWriter.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	ru.Debugf("all closed. uploaded / total = %d / %d", ru.UploadedBytes, ru.TotalBytes)

	return nil
}

// Write is our implementation of io.Writer
func (ru *ResumableUpload) Write(p []byte) (int, error) {
	return ru.writeCounter.Write(p)
}

func (ru *ResumableUpload) Debugf(f string, args ...interface{}) {
	ru.consumer.Debugf("[upload %d] %s", ru.id, fmt.Sprintf(f, args...))
}

func (ru *ResumableUpload) VerboseDebugf(f string, args ...interface{}) {
	if resumableVerboseDebug > 0 {
		ru.consumer.Debugf("[upload %d] %s", ru.id, fmt.Sprintf(f, args...))
	}
}

const gcsChunkSize = 256 * 1024 // 256KB

type blockItem struct {
	buf    []byte
	isLast bool
}

func (ru *ResumableUpload) queryStatus() (*http.Response, error) {
	ru.Debugf("querying upload status...")

	req, err := http.NewRequest("PUT", ru.uploadURL, nil)
	if err != nil {
		// does not include HTTP errors, more like golang API usage errors
		return nil, errors.WithStack(err)
	}

	// for resumable uploads of unknown size, the length is unknown,
	// see https://github.com/itchio/butler/issues/71#issuecomment-242938495
	req.Header.Set("content-range", "bytes */*")

	retryCtx := ru.newRetryContext()
	for retryCtx.ShouldTry() {
		res, err := ru.httpClient.Do(req)
		if err != nil {
			ru.Debugf("while querying status of upload: %s", err.Error())
			retryCtx.Retry(err)
			continue
		}

		status := interpretGcsStatusCode(res.StatusCode)
		if status == GcsResume {
			// got what we wanted (Range header, etc.)
			return res, nil
		}

		if status == GcsNeedQuery {
			retryCtx.Retry(errors.Errorf("while querying status, got HTTP %s (%s)", res.Status, status))
			continue
		}

		return nil, fmt.Errorf("while querying status, got HTTP %s (%s)", res.Status, status)
	}

	return nil, fmt.Errorf("gave up on trying to get upload status")
}

func (ru *ResumableUpload) trySendBytes(buf []byte, offset int64, isLast bool) error {
	buflen := int64(len(buf))
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
		// does not include HTTP errors, more like golang API usage errors
		return errors.WithStack(err)
	}

	start := offset
	end := start + buflen - 1
	contentRange := fmt.Sprintf("bytes %d-%d/*", offset, end)

	if isLast {
		contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, end, offset+buflen)
	}

	req.Header.Set("content-range", contentRange)
	req.ContentLength = buflen
	ru.Debugf("uploading %d-%d, last? %v, content-length set to %d", start, end, isLast, req.ContentLength)

	startTime := time.Now()

	res, err := ru.httpClient.Do(req)
	if err != nil {
		ru.Debugf("while uploading %d-%d: \n%s", start, end, err.Error())
		return &netError{err, GcsUnknown}
	}

	ru.Debugf("server replied in %s, with status %s", time.Since(startTime), res.Status)
	for k, v := range res.Header {
		ru.Debugf("[Reply header] %s: %s", k, v)
	}

	if buflen != int64(len(buf)) {
		// see https://github.com/itchio/butler/issues/71#issuecomment-243081797
		return &netError{fmt.Errorf("send buffer size changed while we were uploading"), GcsResume}
	}

	status := interpretGcsStatusCode(res.StatusCode)
	if status == GcsUploadComplete && isLast {
		ru.Debugf("upload complete!")
		return nil
	}

	if status == GcsNeedQuery {
		ru.Debugf("need to query upload status (HTTP %s)", res.Status)
		statusRes, err := ru.queryStatus()
		if err != nil {
			// this happens after we retry the query a few times
			return err
		}

		if statusRes.StatusCode == 308 {
			ru.Debugf("got upload status, trying to resume")
			res = statusRes
			status = GcsResume
		} else {
			status = interpretGcsStatusCode(statusRes.StatusCode)
			err = fmt.Errorf("expected upload status, got HTTP %s (%s) instead", statusRes.Status, status)
			ru.Debugf(err.Error())
			return err
		}
	}

	if status == GcsResume {
		expectedOffset := offset + buflen
		rangeHeader := res.Header.Get("Range")
		if rangeHeader == "" {
			ru.Debugf("commit failed (null range), retrying")
			return &retryError{committedBytes: 0}
		}

		committedRange, err := parseRangeHeader(rangeHeader)
		if err != nil {
			return err
		}

		ru.Debugf("got resume, expectedOffset: %d, committedRange: %s", expectedOffset, committedRange)
		if committedRange.start != 0 {
			return fmt.Errorf("upload failed: beginning not committed somehow (committed range: %s)", committedRange)
		}

		if committedRange.end == expectedOffset {
			ru.Debugf("commit succeeded (%d blocks stored)", buflen/gcsChunkSize)
			return nil
		} else {
			committedBytes := committedRange.end - offset
			if committedBytes < 0 {
				return fmt.Errorf("upload failed: committed negative bytes somehow (committed range: %s, expectedOffset: %d)", committedRange, expectedOffset)
			}

			if committedBytes > 0 {
				ru.Debugf("commit partially succeeded (committed %d / %d byte, %d blocks)", committedBytes, buflen, committedBytes/gcsChunkSize)
				return &retryError{committedBytes}
			} else {
				ru.Debugf("commit failed (retrying %d blocks)", buflen/gcsChunkSize)
				return &retryError{committedBytes}
			}
		}
	}

	return fmt.Errorf("got HTTP %d (%s)", res.StatusCode, status)
}

func (ru *ResumableUpload) newRetryContext() *retrycontext.Context {
	return retrycontext.New(retrycontext.Settings{
		MaxTries: resumableMaxRetries,
		Consumer: ru.consumer,
	})
}

func (ru *ResumableUpload) uploadChunks(reader io.Reader, done chan bool, errs chan error) {
	var offset int64 = 0

	var maxSendBuf = ru.MaxChunkGroup * gcsChunkSize // 16MB
	sendBuf := make([]byte, 0, maxSendBuf)
	reqBlocks := make(chan blockItem, ru.MaxChunkGroup)

	// when closed, all subtasks should abort
	canceller := make(chan bool)

	sendBytes := func(buf []byte, isLast bool) error {
		retryCtx := ru.newRetryContext()

		for retryCtx.ShouldTry() {
			err := ru.trySendBytes(buf, offset, isLast)
			if err != nil {
				if ne, ok := err.(*netError); ok {
					retryCtx.Retry(ne)
					continue
				} else if re, ok := err.(*retryError); ok {
					offset += re.committedBytes
					buf = buf[re.committedBytes:]
					retryCtx.Retry(errors.Errorf("Having troubles uploading some blocks"))
					continue
				} else {
					return errors.WithStack(err)
				}
			} else {
				offset += int64(len(buf))
				return nil
			}
		}

		return fmt.Errorf("Too many errors, giving up.")
	}

	subDone := make(chan bool)
	subErrs := make(chan error)

	ru.Debugf("sender: starting up, upload URL: %s", ru.uploadURL)

	go func() {
		isLast := false

		// last block needs special treatment (different headers, etc.)
		for !isLast {
			sendBuf = sendBuf[:0]

			// fill send buffer with as many blocks as are
			// available. if none are available, wait for one.
			for len(sendBuf) < maxSendBuf && !isLast {
				var item blockItem
				if len(sendBuf) == 0 {
					ru.VerboseDebugf("sender: doing blocking receive")
					select {
					case item = <-reqBlocks:
						// done waiting, got one, can resume upload now
					case <-canceller:
						ru.Debugf("sender: cancelled (from blocking receive)")
						return
					}
				} else {
					ru.VerboseDebugf("sender: doing non-blocking receive")
					select {
					case item = <-reqBlocks:
						// cool
					case <-canceller:
						ru.Debugf("sender: cancelled (from non-blocking receive)")
						return
					default:
						ru.VerboseDebugf("sent faster than scanned, uploading smaller chunk")
						break
					}
				}

				// if the last item is in sendBuf, sendBuf is the last upload we'll
				// do (save for retries)
				if item.isLast {
					isLast = true
				}

				sendBuf = append(sendBuf, item.buf...)
			}

			if len(sendBuf) > 0 {
				err := sendBytes(sendBuf, isLast)
				if err != nil {
					ru.Debugf("sender: send error, bailing out")
					subErrs <- errors.WithStack(err)
					return
				}
			}
		}

		subDone <- true
		ru.Debugf("sender: all done")
	}()

	// use a bufio.Scanner to break input into blocks of gcsChunkSize
	// at most. last block might be smaller. see splitfunc.go
	s := bufio.NewScanner(reader)
	s.Buffer(make([]byte, gcsChunkSize), 0)
	s.Split(splitfunc.New(gcsChunkSize))

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

	// using two buffers lets us detect EOF even when the last block
	// is an exact multiple of gcsChunkSize - the `for := range` will
	// end and we'll have the last block left in buf1
	buf1 := make([]byte, 0, gcsChunkSize)
	buf2 := make([]byte, 0, gcsChunkSize)

	go func() {
		for scannedBuf := range scannedBufs {
			buf2 = append(buf2[:0], buf1...)
			buf1 = append(buf1[:0], scannedBuf...)
			usedBufs <- true

			// on first iteration, buf2 is still empty.
			if len(buf2) > 0 {
				select {
				case reqBlocks <- blockItem{buf: append([]byte{}, buf2...), isLast: false}:
					// sender received the block, we can keep going
				case <-canceller:
					ru.Debugf("scan cancelled (3)")
					return
				}
			}
		}

		err := s.Err()
		if err != nil {
			ru.Debugf("scanner error :(")
			subErrs <- errors.WithStack(err)
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
			// any error that travels this far up cancels the whole upload
			close(canceller)
			errs <- errors.WithStack(err)
			return
		}
	}

	done <- true
	ru.Debugf("done sent!")
}
