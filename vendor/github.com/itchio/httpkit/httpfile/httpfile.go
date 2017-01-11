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
	"strconv"
	"strings"
	"sync"
	"syscall"
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

const maxRenewals = 5

// ErrNotFound is returned when the HTTP server returns 404 - it's not considered a temporary error
var ErrNotFound = errors.New("HTTP file not found on server")

var ErrTooManyRenewals = errors.New("Giving up, getting too many renewals. Try again later or contact support.")

// HTTPFile allows accessing a file served by an HTTP server as if it was local
// (for random-access reading purposes, not writing)
type HTTPFile struct {
	getURL        GetURLFunc
	needsRenewal  NeedsRenewalFunc
	client        *http.Client
	retrySettings *retrycontext.Settings

	Log      LogFunc
	LogLevel int

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

// DefaultReaderStaleThreshold is the duration after which HTTPFile's readers
// are considered stale, and are closed instead of reused. It's set to 10 seconds.
const DefaultReaderStaleThreshold = time.Second * time.Duration(10)

const DefaultLogLevel = 1

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

type NeedsRenewalError struct {
	url string
}

func (nre *NeedsRenewalError) Error() string {
	return "url has expired and needs renewal"
}

type ServerError struct {
	host    string
	message string
}

func (se *ServerError) Error() string {
	return fmt.Sprintf("server error: for host %s: %s", se.host, se.message)
}

func (hr *httpReader) Connect() error {
	hf := hr.file

	if hr.body != nil {
		err := hr.body.Close()
		if err != nil {
			return err
		}

		hr.body = nil
		hr.reader = nil
	}

	tryURL := func(urlStr string) error {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return err
		}

		byteRange := fmt.Sprintf("bytes=%d-", hr.offset)
		req.Header.Set("Range", byteRange)

		res, err := hf.client.Do(req)
		if err != nil {
			return err
		}
		hf.log("Connect.tryURL: HTTP %d", res.StatusCode)

		if res.StatusCode == 200 && hr.offset > 0 {
			hf.log("Connect.tryURL: HTTP range header not supported")
			defer res.Body.Close()
			return &ServerError{host: req.Host, message: fmt.Sprintf("HTTP Range header not supported")}
		}

		if res.StatusCode/100 != 2 {
			hf.log("Connect.tryURL: HTTP non-200, reading error")
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				body = []byte("could not read error body")
				err = nil
			}

			if hf.needsRenewal(res, body) {
				hf.log("Connect.tryURL: needs renewal")
				return &NeedsRenewalError{url: urlStr}
			}

			hf.log("Connect.tryURL: no renewal")
			return &ServerError{host: req.Host, message: fmt.Sprintf("HTTP %d received, body = %s", res.StatusCode, string(body))}
		}

		hr.reader = bufio.NewReaderSize(res.Body, int(maxDiscard))
		hr.body = res.Body
		return nil
	}

	urlStr := hf.getCurrentURL()

	retryCtx := hf.newRetryContext()
	renewalTries := 0

	for retryCtx.ShouldTry() {
		fmt.Printf("Trying url...\n")
		hf.log("Connect: trying url...")
		err := tryURL(urlStr)
		if err != nil {
			if _, ok := err.(*NeedsRenewalError); ok {
				renewalTries++
				if renewalTries >= maxRenewals {
					return ErrTooManyRenewals
				}

				hf.log("Connect: got renew: %s", err.Error())
				renewRetryCtx := hf.newRetryContext()

				for renewRetryCtx.ShouldTry() {
					urlStr, err = hf.renewURL()
					if err != nil {
						if hf.shouldRetry(err) {
							hf.log("Connect.renew: got retriable error: %s", err.Error())
							renewRetryCtx.Retry(err.Error())
							continue
						} else {
							hf.log("Connect.renew: got non-retriable error: %s", err.Error())
							return err
						}
					}
					break
				}
				continue
			} else if hf.shouldRetry(err) {
				hf.log("Connect: got retriable error: %s", err.Error())
				retryCtx.Retry(err.Error())
				continue
			} else {
				hf.log("Connect: got non-renew, non-retriable error: %s", err.Error())
				return err
			}
		}
		break
	}

	hf.log("Connect: connected!")
	return nil
}

func (hr *httpReader) Close() error {
	if hr.body != nil {
		err := hr.body.Close()
		hr.body = nil

		if err != nil {
			return err
		}
	}

	return nil
}

var _ io.Seeker = (*HTTPFile)(nil)
var _ io.Reader = (*HTTPFile)(nil)
var _ io.ReaderAt = (*HTTPFile)(nil)
var _ io.Closer = (*HTTPFile)(nil)

// Settings allows passing additional settings to an HTTPFile
type Settings struct {
	Client        *http.Client
	RetrySettings *retrycontext.Settings
}

// New returns a new HTTPFile. Note that it differs from os.Open in that it does a first request
// to determine the remote file's size. If that fails (after retries), an error will be returned.
func New(getURL GetURLFunc, needsRenewal NeedsRenewalFunc, settings *Settings) (*HTTPFile, error) {
	client := settings.Client
	if client == nil {
		client = http.DefaultClient
	}

	retryCtx := retrycontext.NewDefault()
	if settings.RetrySettings != nil {
		retryCtx.Settings = *settings.RetrySettings
	}

	hf := &HTTPFile{
		getURL:        getURL,
		retrySettings: &retryCtx.Settings,
		needsRenewal:  needsRenewal,
		client:        client,

		readers: make(map[string]*httpReader),

		ReaderStaleThreshold: DefaultReaderStaleThreshold,
		LogLevel:             DefaultLogLevel,
	}

	renewalTries := 0

	for retryCtx.ShouldTry() {
		urlStr, err := getURL()
		if err != nil {
			// this assumes getURL does its own retrying
			return nil, err
		}

		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			// can't recover from a bad url
			return nil, err
		}

		// This used to be `HEAD`, but some servers (looking at you Amazon S3)
		// didn't like it.

		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			// internal error
			return nil, err
		}

		req.Header.Set("Range", "bytes=0-0")

		res, err := client.Do(req)
		if err != nil {
			if hf.shouldRetry(err) {
				// we can recover from some client errors
				// (example: temporarily offline, DNS failure, etc.)
				retryCtx.Retry(err.Error())
				continue
			} else {
				return nil, err
			}
		}

		err = res.Body.Close()
		if err != nil {
			return nil, err
		}

		if res.StatusCode != 206 && res.StatusCode != 200 {
			if res.StatusCode == 404 {
				// no need to retry - it's not coming back
				return nil, errors.Wrap(ErrNotFound, 1)
			}

			body, _ := ioutil.ReadAll(res.Body)
			if needsRenewal(res, body) {
				// don't sleep for renewal
				hf.log("Initial request needs renewal (HTTP %d). Good start, good start.", res.StatusCode)

				renewalTries++
				if renewalTries >= maxRenewals {
					return nil, ErrTooManyRenewals
				}
				continue
			}

			if res.StatusCode == 429 || res.StatusCode/100 == 5 {
				retryCtx.Retry(fmt.Sprintf("HTTP %d (retrying)", res.StatusCode))
				continue
			}

			return nil, fmt.Errorf("Expected HTTP 206, got HTTP %d, not retrying", res.StatusCode)
		}

		var totalBytes int64

		if res.StatusCode == 206 {
			rangeHeader := res.Header.Get("content-range")
			rangeTokens := strings.Split(rangeHeader, "/")
			totalBytesStr := rangeTokens[len(rangeTokens)-1]
			totalBytes, err = strconv.ParseInt(totalBytesStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("Could not parse file size: %s", err.Error())
			}
		} else if res.StatusCode == 200 {
			totalBytes = res.ContentLength
		}

		hf.currentURL = urlStr
		hf.name = parsedURL.Path
		hf.size = totalBytes
		return hf, nil
	}

	return nil, fmt.Errorf("Could not access remote file. Last error: %s", retryCtx.LastMessage)
}

func (hf *HTTPFile) newRetryContext() *retrycontext.Context {
	retryCtx := retrycontext.NewDefault()
	if hf.retrySettings != nil {
		retryCtx.Settings = *hf.retrySettings
	}
	return retryCtx
}

// NumReaders returns the number of connections currently used by the httpfile
// to serve ReadAt calls
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
			hf.log2("borrow: for %d, re-using %d by discarding %d bytes", offset, reader.offset, bestDiff)

			// XXX: not int64-clean
			_, err := reader.Discard(int(bestDiff))
			if err != nil {
				if hf.shouldRetry(err) {
					hf.log2("borrow: for %d, discard failed because of retriable error, reconnecting", offset)
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

// Stat returns an os.FileInfo for this particular file. Only the Size()
// method is useful, the rest is default values.
func (hf *HTTPFile) Stat() (os.FileInfo, error) {
	return &httpFileInfo{hf}, nil
}

// Seek the read head within the file - it's instant and never returns an
// error, except if whence is one of os.SEEK_SET, os.SEEK_END, or os.SEEK_CUR.
// If an invalid offset is given, it will be truncated to a valid one, between
// [0,size).
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
	hf.log2("Read(%d)", len(data))
	bytesRead, err := hf.readAt(data, hf.offset)
	hf.offset += int64(bytesRead)
	hf.log2("Read(%d) = %d, %+v", len(data), bytesRead, err != nil)
	return bytesRead, err
}

// ReadAt reads len(data) byte from the remote file at offset.
// It returns the number of bytes read, and an error. In case of temporary
// network errors or timeouts, it will retry with truncated exponential backoff
// according to RetrySettings
func (hf *HTTPFile) ReadAt(data []byte, offset int64) (int, error) {
	hf.log2("ReadAt(%d, %d)", len(data), offset)
	n, err := hf.readAt(data, offset)
	hf.log2("ReadAt(%d, %d) = %d, %+v", len(data), offset, n, err != nil)
	return n, err
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
			if hf.shouldRetry(err) {
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

func (hf *HTTPFile) shouldRetry(err error) bool {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		hf.log("shouldRetry: retrying unexpected EOF")
		return true
	} else if opError, ok := err.(*net.OpError); ok {
		// examples (win): "read tcp [...]: wsarecv: An established connection was aborted by the software in your host machine"
		if opError.Timeout() ||
			opError.Temporary() ||
			opError.Err.Error() == syscall.ECONNRESET.Error() ||
			strings.HasPrefix(opError.Err.Error(), "read tcp") {
			hf.log("shouldRetry: retrying net.OpError %s, nested error: %s", err.Error(), opError.Err.Error())
			return true
		} else {
			hf.log("shouldRetry: bailing on net.OpError %s, nested error: %s", err.Error(), opError.Err.Error())
		}
	} else if urlError, ok := err.(*url.Error); ok {
		// examples: "dial tcp: [...] on port 53: server misbehaving"
		// examples: "dial tcp: [...] on port 53: timed out"
		if urlError.Timeout() ||
			urlError.Temporary() ||
			errors.Is(urlError.Err, io.EOF) ||
			strings.HasPrefix(urlError.Err.Error(), "dial tcp") {
			hf.log("shouldRetry: retrying url.Error %s, nested error: %s", err.Error(), urlError.Err.Error())
			return true
		} else {
			hf.log("shouldRetry: bailing on url.Error %s, nested error: %s", err.Error(), urlError.Err.Error())
		}
	} else {
		hf.log("shouldRetry: bailing on unknown error %s", err.Error())
	}

	return false
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

// Close closes all connections to the distant http server used by this HTTPFile
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

func (hf *HTTPFile) log2(format string, args ...interface{}) {
	if hf.LogLevel < 2 {
		return
	}

	if hf.Log == nil {
		return
	}

	hf.Log(fmt.Sprintf(format, args...))
}
