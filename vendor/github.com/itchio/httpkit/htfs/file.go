package htfs

import (
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	goerrors "errors"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/httpkit/neterr"
	"github.com/itchio/httpkit/retrycontext"
	"github.com/pkg/errors"
)

var forbidBacktracking = os.Getenv("HFS_NO_BACKTRACK") == "1"
var dumpStats = os.Getenv("HFS_DUMP_STATS") == "1"

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
var ErrNotFound = goerrors.New("HTTP file not found on server")

// ErrTooManyRenewals is returned when we keep calling the GetURLFunc but it
// immediately return an errors marked as renewal-related by NeedsRenewalFunc.
// This can happen when servers are misconfigured.
var ErrTooManyRenewals = goerrors.New("Giving up, getting too many renewals. Try again later or contact support.")

type hstats struct {
	connectionWait time.Duration
	connections    int
	renews         int

	fetchedBytes int64
	numCacheMiss int64

	cachedBytes int64
	numCacheHit int64
}

var idSeed int64 = 1
var idMutex sync.Mutex

// File allows accessing a file served by an HTTP server as if it was local
// (for random-access reading purposes, not writing)
type File struct {
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

	readers map[string]*conn
	lock    sync.Mutex

	currentURL string
	urlMutex   sync.Mutex
	header     http.Header
	requestURL *url.URL

	stats *hstats

	ForbidBacktracking bool
}

const defaultLogLevel = 1

// defaultReaderStaleThreshold is the duration after which File's readers
// are considered stale, and are closed instead of reused. It's set to 10 seconds.
const defaultReaderStaleThreshold = time.Second * time.Duration(10)

var _ io.Seeker = (*File)(nil)
var _ io.Reader = (*File)(nil)
var _ io.ReaderAt = (*File)(nil)
var _ io.Closer = (*File)(nil)

// Settings allows passing additional settings to an File
type Settings struct {
	Client             *http.Client
	RetrySettings      *retrycontext.Settings
	Log                LogFunc
	LogLevel           int
	ForbidBacktracking bool
}

// Open returns a new htfs.File. Note that it differs from os.Open in that it does a first request
// to determine the remote file's size. If that fails (after retries), an error will be returned.
func Open(getURL GetURLFunc, needsRenewal NeedsRenewalFunc, settings *Settings) (*File, error) {
	client := settings.Client
	if client == nil {
		client = http.DefaultClient
	}

	retryCtx := retrycontext.NewDefault()
	if settings.RetrySettings != nil {
		retryCtx.Settings = *settings.RetrySettings
	}

	hf := &File{
		getURL:        getURL,
		retrySettings: &retryCtx.Settings,
		needsRenewal:  needsRenewal,
		client:        client,
		name:          "<remote file>",

		readers: make(map[string]*conn),
		stats:   &hstats{},

		ReaderStaleThreshold: defaultReaderStaleThreshold,
		LogLevel:             defaultLogLevel,
		ForbidBacktracking:   forbidBacktracking,
	}
	hf.Log = settings.Log

	if settings.LogLevel != 0 {
		hf.LogLevel = settings.LogLevel
	}
	if settings.ForbidBacktracking {
		hf.ForbidBacktracking = true
	}

	urlStr, err := getURL()
	if err != nil {
		return nil, errors.WithMessage(normalizeError(err), "htfs.Open (getting URL)")
	}
	hf.currentURL = urlStr

	hr, err := hf.borrowReader(0)
	if err != nil {
		return nil, errors.WithMessage(normalizeError(err), "htfs.Open (initial request)")
	}
	hf.returnReader(hr)

	hf.requestURL = hr.requestURL

	if hr.statusCode == 206 {
		rangeHeader := hr.header.Get("content-range")
		rangeTokens := strings.Split(rangeHeader, "/")
		totalBytesStr := rangeTokens[len(rangeTokens)-1]
		hf.size, err = strconv.ParseInt(totalBytesStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Could not parse file size: %s", err.Error())
		}
	} else if hr.statusCode == 200 {
		hf.size = hr.contentLength
	}

	// we have to use requestURL because we want the URL after
	// redirect (for hosts like sourceforge)
	pathTokens := strings.Split(hf.requestURL.Path, "/")
	hf.name = pathTokens[len(pathTokens)-1]

	dispHeader := hr.header.Get("content-disposition")
	if dispHeader != "" {
		_, mimeParams, err := mime.ParseMediaType(dispHeader)
		if err == nil {
			filename := mimeParams["filename"]
			if filename != "" {
				hf.name = filename
			}
		}
	}

	return hf, nil
}

func (hf *File) newRetryContext() *retrycontext.Context {
	retryCtx := retrycontext.NewDefault()
	if hf.retrySettings != nil {
		retryCtx.Settings = *hf.retrySettings
	}
	return retryCtx
}

// NumReaders returns the number of connections currently used by the File
// to serve ReadAt calls
func (hf *File) NumReaders() int {
	hf.lock.Lock()
	defer hf.lock.Unlock()

	return len(hf.readers)
}

func (hf *File) borrowReader(offset int64) (*conn, error) {
	if hf.size > 0 && offset >= hf.size {
		return nil, io.EOF
	}

	var bestReader string
	var bestDiff int64 = math.MaxInt64

	var bestBackReader string
	var bestBackDiff int64 = math.MaxInt64

	for _, reader := range hf.readers {
		if reader.Stale() {
			delete(hf.readers, reader.id)

			err := reader.Close()
			if err != nil {
				return nil, err
			}
			continue
		}

		diff := offset - reader.Offset()
		if diff < 0 && -diff < maxDiscard && -diff <= reader.Cached() {
			if -diff < bestBackDiff {
				bestBackReader = reader.id
				bestBackDiff = -diff
			}
		}

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

		// clear backtrack if any
		reader.Backtrack(0)

		// discard if needed
		if bestDiff > 0 {
			hf.log2("[%9d-%9d] (Borrow) %d --> %d (%s)", offset, offset, reader.Offset(), reader.Offset()+bestDiff, reader.id)

			err := reader.Discard(bestDiff)
			if err != nil {
				if hf.shouldRetry(err) {
					hf.log2("[%9d-] (Borrow) discard failed, reconnecting", offset)
					err = reader.Connect(offset)
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

	if !hf.ForbidBacktracking && bestBackReader != "" {
		// re-use!
		reader := hf.readers[bestBackReader]
		delete(hf.readers, bestBackReader)

		hf.log2("[%9d-%9d] (Borrow) %d <-- %d (%s)", offset, offset, reader.Offset()-bestBackDiff, reader.Offset(), reader.id)

		// backtrack as needed
		err := reader.Backtrack(bestBackDiff)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return reader, nil
	}

	// provision a new reader
	hf.log("[%9d-%9d] (Borrow) new connection", offset, offset)

	id := generateID()
	reader := &conn{
		file:      hf,
		id:        fmt.Sprintf("reader-%d", id),
		touchedAt: time.Now(),
	}

	err := reader.Connect(offset)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func (hf *File) returnReader(reader *conn) {
	// TODO: enforce max idle readers ?

	reader.touchedAt = time.Now()
	hf.readers[reader.id] = reader
}

func (hf *File) getCurrentURL() string {
	hf.urlMutex.Lock()
	defer hf.urlMutex.Unlock()

	return hf.currentURL
}

func (hf *File) renewURL() (string, error) {
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
func (hf *File) Stat() (os.FileInfo, error) {
	return &FileInfo{hf}, nil
}

// Seek the read head within the file - it's instant and never returns an
// error, except if whence is one of os.SEEK_SET, os.SEEK_END, or os.SEEK_CUR.
// If an invalid offset is given, it will be truncated to a valid one, between
// [0,size).
func (hf *File) Seek(offset int64, whence int) (int64, error) {
	hf.lock.Lock()
	defer hf.lock.Unlock()

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

func (hf *File) Read(buf []byte) (int, error) {
	hf.lock.Lock()
	defer hf.lock.Unlock()

	initialOffset := hf.offset
	bytesRead, err := hf.readAt(buf, hf.offset)
	hf.offset += int64(bytesRead)

	if hf.LogLevel >= 2 {
		bytesWanted := int64(len(buf))
		start := initialOffset
		end := initialOffset + bytesWanted
		hf.log2("[%9d-%9d] (Read) %d/%d %v", start, end, bytesRead, bytesWanted, err)
	}
	return bytesRead, err
}

// ReadAt reads len(buf) byte from the remote file at offset.
// It returns the number of bytes read, and an error. In case of temporary
// network errors or timeouts, it will retry with truncated exponential backoff
// according to RetrySettings
func (hf *File) ReadAt(buf []byte, offset int64) (int, error) {
	hf.lock.Lock()
	defer hf.lock.Unlock()

	bytesRead, err := hf.readAt(buf, offset)

	if hf.LogLevel >= 2 {
		bytesWanted := int64(len(buf))
		start := offset
		end := offset + bytesWanted

		var readDesc string
		if bytesWanted == int64(bytesRead) {
			readDesc = "full"
		} else if bytesRead == 0 {
			readDesc = fmt.Sprintf("partial (%d of %d)", bytesRead, bytesWanted)
		} else {
			readDesc = "zero"
		}
		if err != nil {
			readDesc += fmt.Sprintf(", with err %v", err)
		}
		hf.log2("[%9d-%9d] (ReadAt) %s", start, end, readDesc)
	}
	return bytesRead, err
}

func (hf *File) readAt(data []byte, offset int64) (int, error) {
	buflen := len(data)
	if buflen == 0 {
		return 0, nil
	}

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
				err = reader.Connect(reader.Offset())
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

func (hf *File) shouldRetry(err error) bool {
	if errors.Cause(err) == io.EOF {
		// don't retry EOF, it's a perfectly expected error
		return false
	}

	if neterr.IsNetworkError(err) {
		hf.log("Retrying: %v", err)
		return true
	}

	if se, ok := errors.Cause(err).(*ServerError); ok {
		switch se.StatusCode {
		case 429: /* Too Many Requests */
			return true
		case 500: /* Internal Server Error */
			return true
		case 502: /* Bad Gateway */
			return true
		case 503: /* Service Unavailable */
			return true
		}
	}

	hf.log("Bailing on error: %v", err)
	return false
}

func isHTTPStatus(err error, statusCode int) bool {
	if se, ok := errors.Cause(err).(*ServerError); ok {
		return se.StatusCode == statusCode
	}
	return false
}

func normalizeError(err error) error {
	if isHTTPStatus(err, 404) {
		return ErrNotFound
	}
	return err
}

func (hf *File) closeAllReaders() error {
	for id, reader := range hf.readers {
		err := reader.Close()
		if err != nil {
			return err
		}

		delete(hf.readers, id)
	}

	return nil
}

// Close closes all connections to the distant http server used by this File
func (hf *File) Close() error {
	hf.lock.Lock()
	defer hf.lock.Unlock()

	if hf.closed {
		return nil
	}

	if dumpStats {
		log.Printf("========= File stats ==============")
		log.Printf("= total connections: %d", hf.stats.connections)
		log.Printf("= total renews: %d", hf.stats.renews)
		log.Printf("= time spent connecting: %s", hf.stats.connectionWait)
		size := hf.size
		perc := 0.0
		percCached := 0.0
		if size != 0 {
			perc = float64(hf.stats.fetchedBytes) / float64(size) * 100.0
		}
		allReads := hf.stats.fetchedBytes + hf.stats.cachedBytes
		percCached = float64(hf.stats.cachedBytes) / float64(allReads) * 100.0

		log.Printf("= total bytes fetched: %s / %s (%.2f%%)", humanize.IBytes(uint64(hf.stats.fetchedBytes)), humanize.IBytes(uint64(size)), perc)
		log.Printf("= total bytes served from cache: %s (%.2f%% of all served bytes)", humanize.IBytes(uint64(hf.stats.cachedBytes)), percCached)

		hitRate := float64(hf.stats.numCacheHit) / float64(hf.stats.numCacheHit+hf.stats.numCacheMiss) * 100.0
		log.Printf("= cache hit rate: %.2f%%", hitRate)
		log.Printf("========================================")
	}

	err := hf.closeAllReaders()
	if err != nil {
		return err
	}

	hf.closed = true

	return nil
}

func (hf *File) log(format string, args ...interface{}) {
	if hf.Log == nil {
		return
	}

	hf.Log(fmt.Sprintf(format, args...))
}

func (hf *File) log2(format string, args ...interface{}) {
	if hf.LogLevel < 2 {
		return
	}

	if hf.Log == nil {
		return
	}

	hf.Log(fmt.Sprintf(format, args...))
}

// GetHeader returns the header the server responded
// with on our initial request. It may contain checksums
// which could be used for integrity checking.
func (hf *File) GetHeader() http.Header {
	return hf.header
}

// GetRequestURL returns the first good URL File
// made a request to.
func (hf *File) GetRequestURL() *url.URL {
	return hf.requestURL
}

func generateID() int64 {
	idMutex.Lock()
	defer idMutex.Unlock()

	id := idSeed
	idSeed++
	return id
}
