package httpfile

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/httpkit/retrycontext"
	uuid "github.com/satori/go.uuid"
)

// A GetURLFunc returns a URL we can download the resource from.
// It's handy to have this as a function rather than a constant for signed expiring URLs
type GetURLFunc func() (urlString string, err error)

// A NeedsRenewalFunc analyzes an HTTP response and returns true if it needs to be renewed
type NeedsRenewalFunc func(res *http.Response, body []byte) bool

// A LogFunc prints debug message
type LogFunc func(msg string)

// amount we're willing to download and throw away
const maxDiscard int64 = 1 * 1024 * 1024 // 1MB

var ErrNotFound = errors.New("HTTP file not found on server")

type HTTPFile struct {
	getURL        GetURLFunc
	needsRenewal  NeedsRenewalFunc
	client        *http.Client
	retrySettings *retrycontext.Settings

	Log LogFunc

	name   string
	size   int64
	offset int64 // for io.ReadSeeker

	ReaderStaleThreshold time.Duration

	closed bool

	readers      map[string]*httpReader
	readersMutex sync.Mutex

	currentURL string
	urlMutex   sync.Mutex
}

type httpReader struct {
	file      *HTTPFile
	id        string
	touchedAt time.Time
	offset    int64
	body      io.ReadCloser
	reader    *bufio.Reader
}

const DefaultReaderStaleThreshold = time.Second * time.Duration(10)

func (hr *httpReader) Stale() bool {
	return time.Since(hr.touchedAt) > hr.file.ReaderStaleThreshold
}

func (hr *httpReader) Read(data []byte) (int, error) {
	hr.touchedAt = time.Now()
	readBytes, err := hr.reader.Read(data)
	hr.offset += int64(readBytes)

	if err != nil {
		return readBytes, err
	}
	return readBytes, nil
}

func (hr *httpReader) Discard(n int) (int, error) {
	hr.touchedAt = time.Now()
	discarded, err := hr.reader.Discard(n)
	hr.offset += int64(discarded)

	if err != nil {
		return discarded, err
	}
	return discarded, nil
}

func (hr *httpReader) Connect() error {
	if hr.body != nil {
		err := hr.body.Close()
		if err != nil {
			return err
		}

		hr.body = nil
		hr.reader = nil
	}

	tryUrl := func(urlStr string) (bool, error) {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return false, err
		}

		byteRange := fmt.Sprintf("bytes=%d-", hr.offset)
		req.Header.Set("Range", byteRange)

		res, err := hr.file.client.Do(req)
		if err != nil {
			return false, err
		}
		hr.file.log("did request, status %d", res.StatusCode)

		if res.StatusCode == 200 && hr.offset > 0 {
			defer res.Body.Close()

			err = fmt.Errorf("HTTP Range header not supported by %s, bailing out", req.Host)
			return false, err
		}

		if res.StatusCode/100 != 2 {
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				body = []byte("could not read error body")
				err = nil
			}

			if hr.file.needsRenewal(res, body) {
				return true, nil
			}

			err = fmt.Errorf("HTTP %d returned by %s (%s), bailing out", res.StatusCode, req.Host, string(body))
			return false, err
		}

		hr.reader = bufio.NewReaderSize(res.Body, int(maxDiscard))
		hr.body = res.Body
		return false, nil
	}

	urlStr := hr.file.getCurrentURL()
	shouldRenew, err := tryUrl(urlStr)
	if err != nil {
		return err
	}

	if shouldRenew {
		urlStr, err = hr.file.renewURL()
		if err != nil {
			return err
		}

		shouldRenew, err = tryUrl(urlStr)
		if err != nil {
			return err
		}

		if shouldRenew {
			return fmt.Errorf("getting expired URLs from URL source (timezone issue?)")
		}
	}

	return nil
}

func (hr *httpReader) Close() error {
	err := hr.body.Close()

	if err != nil {
		return err
	}
	return nil
}

var _ io.Seeker = (*HTTPFile)(nil)
var _ io.Reader = (*HTTPFile)(nil)
var _ io.ReaderAt = (*HTTPFile)(nil)
var _ io.Closer = (*HTTPFile)(nil)

type Settings struct {
	Client        *http.Client
	RetrySettings *retrycontext.Settings
}

func New(getURL GetURLFunc, needsRenewal NeedsRenewalFunc, settings *Settings) (*HTTPFile, error) {
	client := settings.Client
	if client == nil {
		client = http.DefaultClient
	}

	retryCtx := retrycontext.NewDefault()
	if settings.RetrySettings != nil {
		retryCtx.Settings = *settings.RetrySettings
	}

	for retryCtx.ShouldTry() {
		urlStr, err := getURL()
		if err != nil {
			// this assumes getURL does its own retrying
			return nil, err
		}

		parsedUrl, err := url.Parse(urlStr)
		if err != nil {
			// can't recover from a bad url
			return nil, err
		}

		req, err := http.NewRequest("HEAD", urlStr, nil)
		if err != nil {
			// internal error
			return nil, err
		}

		res, err := client.Do(req)
		if err != nil {
			// we can recover from some client errors
			// (example: temporarily offline, DNS failure, etc.)
			retryCtx.Retry(err.Error())
			continue
		}

		if res.StatusCode != 200 {
			if res.StatusCode == 404 {
				// no need to retry - it's not coming back
				return nil, errors.Wrap(ErrNotFound, 1)
			}

			body, _ := ioutil.ReadAll(res.Body)
			if needsRenewal(res, body) {
				retryCtx.Retry(fmt.Sprintf("HTTP %d (needs renewal)", res.StatusCode))
				continue
			}

			if res.StatusCode == 429 || res.StatusCode/100 == 5 {
				retryCtx.Retry(fmt.Sprintf("HTTP %d (retrying)", res.StatusCode))
				continue
			}

			return nil, fmt.Errorf("Expected HTTP 200, got HTTP %d, not retrying", res.StatusCode)
		}

		hf := &HTTPFile{
			currentURL:    urlStr,
			getURL:        getURL,
			retrySettings: &retryCtx.Settings,
			needsRenewal:  needsRenewal,
			client:        client,

			name:    parsedUrl.Path,
			size:    res.ContentLength,
			readers: make(map[string]*httpReader),

			ReaderStaleThreshold: DefaultReaderStaleThreshold,
		}
		return hf, nil
	}

	return nil, fmt.Errorf("Could not access remote file. Last error: %s", retryCtx.LastMessage)
}

func (hf *HTTPFile) NumReaders() int {
	return len(hf.readers)
}

func (hf *HTTPFile) borrowReader(offset int64) (*httpReader, error) {
	hf.readersMutex.Lock()
	defer hf.readersMutex.Unlock()

	var bestReader string
	var bestDiff int64 = math.MaxInt64

	for _, reader := range hf.readers {
		if reader.Stale() {
			delete(hf.readers, reader.id)

			err := reader.Close()
			if err != nil {
				return nil, err
			}
			continue
		}

		diff := offset - reader.offset
		if diff >= 0 && diff < maxDiscard {
			if diff < bestDiff {
				bestReader = reader.id
				bestDiff = diff
			}
		}
	}

	if bestReader != "" {
		// re-use!
		reader := hf.readers[bestReader]
		delete(hf.readers, bestReader)

		// discard if needed
		if bestDiff > 0 {
			hf.log("borrow: for %d, re-using %d by discarding %d bytes", offset, reader.offset, bestDiff)

			// XXX: not int64-clean
			_, err := reader.Discard(int(bestDiff))
			if err != nil {
				if shouldRetry(err) {
					hf.log("borrow: for %d, discard failed because of retriable error, reconnecting", offset)
					reader.offset = offset
					err = reader.Connect()
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		}

		return reader, nil
	}

	// provision a new reader
	hf.log("borrow: making fresh for offset %d", offset)

	reader := &httpReader{
		file:      hf,
		id:        uuid.NewV4().String(),
		touchedAt: time.Now(),
		offset:    offset,
	}

	err := reader.Connect()
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (hf *HTTPFile) returnReader(reader *httpReader) {
	hf.readersMutex.Lock()
	defer hf.readersMutex.Unlock()

	// TODO: enforce max idle readers ?

	reader.touchedAt = time.Now()
	hf.readers[reader.id] = reader
}

func (hf *HTTPFile) getCurrentURL() string {
	hf.urlMutex.Lock()
	defer hf.urlMutex.Unlock()

	return hf.currentURL
}

func (hf *HTTPFile) renewURL() (string, error) {
	hf.urlMutex.Lock()
	defer hf.urlMutex.Unlock()

	urlStr, err := hf.getURL()
	if err != nil {
		return "", err
	}

	hf.currentURL = urlStr
	return hf.currentURL, nil
}

func (hf *HTTPFile) Stat() (os.FileInfo, error) {
	return &httpFileInfo{hf}, nil
}

func (hf *HTTPFile) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case os.SEEK_SET:
		newOffset = offset
	case os.SEEK_END:
		newOffset = hf.size + offset
	case os.SEEK_CUR:
		newOffset = hf.offset + offset
	default:
		return hf.offset, fmt.Errorf("invalid whence value %d", whence)
	}

	if newOffset < 0 {
		newOffset = 0
	}

	if newOffset > hf.size {
		newOffset = hf.size
	}

	hf.offset = newOffset
	return hf.offset, nil
}

func (hf *HTTPFile) Read(data []byte) (int, error) {
	hf.log("Read(%d)", len(data))
	bytesRead, err := hf.readAt(data, hf.offset)
	hf.offset += int64(bytesRead)
	return bytesRead, err
}

func (hf *HTTPFile) ReadAt(data []byte, offset int64) (int, error) {
	hf.log("ReadAt(%d, %d)", len(data), offset)
	return hf.readAt(data, offset)
}

func (hf *HTTPFile) readAt(data []byte, offset int64) (int, error) {
	reader, err := hf.borrowReader(offset)
	if err != nil {
		return 0, err
	}

	defer hf.returnReader(reader)

	totalBytesRead := 0
	bytesToRead := len(data)

	for totalBytesRead < bytesToRead {
		bytesRead, err := reader.Read(data[totalBytesRead:])
		totalBytesRead += bytesRead

		if err != nil {
			if shouldRetry(err) {
				hf.log("Got %s, retrying", err.Error())
				err = reader.Connect()
				if err != nil {
					return totalBytesRead, err
				}
			} else {
				return totalBytesRead, err
			}
		}
	}

	return totalBytesRead, nil
}

func shouldRetry(err error) bool {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	} else if opError, ok := err.(*net.OpError); ok {
		if opError.Timeout() || opError.Temporary() {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

func (hf *HTTPFile) closeAllReaders() error {
	hf.readersMutex.Lock()
	defer hf.readersMutex.Unlock()

	for id, reader := range hf.readers {
		err := reader.Close()
		if err != nil {
			return err
		}

		delete(hf.readers, id)
	}

	return nil
}

func (hf *HTTPFile) Close() error {
	if hf.closed {
		return nil
	}

	err := hf.closeAllReaders()
	if err != nil {
		return err
	}

	hf.closed = true

	return nil
}

func (hf *HTTPFile) log(format string, args ...interface{}) {
	if hf.Log == nil {
		return
	}

	hf.Log(fmt.Sprintf(format, args...))
}
