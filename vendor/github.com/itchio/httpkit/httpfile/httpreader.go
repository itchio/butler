package httpfile

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/itchio/httpkit/httpfile/backtracker"
	"github.com/pkg/errors"
)

// DefaultReaderStaleThreshold is the duration after which HTTPFile's readers
// are considered stale, and are closed instead of reused. It's set to 10 seconds.
const DefaultReaderStaleThreshold = time.Second * time.Duration(10)

type httpReader struct {
	backtracker.Backtracker

	file       *HTTPFile
	id         string
	touchedAt  time.Time
	body       io.ReadCloser
	reader     *bufio.Reader
	currentURL string

	header        http.Header
	requestURL    *url.URL
	statusCode    int
	contentLength int64
}

func (hr *httpReader) Stale() bool {
	return time.Since(hr.touchedAt) > hr.file.ReaderStaleThreshold
}

// *not* thread-safe, httpfile handles the locking
func (hr *httpReader) Connect(offset int64) error {
	hf := hr.file

	if hr.body != nil {
		err := hr.body.Close()
		if err != nil {
			return err
		}

		hr.body = nil
		hr.reader = nil
	}

	retryCtx := hf.newRetryContext()
	renewalTries := 0

	hf.currentURL = hf.getCurrentURL()
	for retryCtx.ShouldTry() {
		startTime := time.Now()
		err := hr.tryConnect(offset)
		if err != nil {
			if _, ok := err.(*NeedsRenewalError); ok {
				renewalTries++
				if renewalTries >= maxRenewals {
					return ErrTooManyRenewals
				}
				hf.log("[%9d-%9d] (Connect) renewing on %v", offset, offset, err)

				err = hr.renewURLWithRetries(offset)
				if err != nil {
					// if we reach this point, we've failed to generate
					// a download URL a bunch of times in a row
					return err
				}
				continue
			} else if hf.shouldRetry(err) {
				hf.log("[%9d-%9d] (Connect) retrying %v", offset, offset, err)
				retryCtx.Retry(err)
				continue
			} else {
				return err
			}
		}

		totalConnDuration := time.Since(startTime)
		hf.log("[%9d-%9d] (Connect) %s", offset, offset, totalConnDuration)
		hf.stats.connections++
		hf.stats.connectionWait += totalConnDuration
		return nil
	}

	return errors.WithMessage(retryCtx.LastError, "httpfile connect")
}

func (hr *httpReader) renewURLWithRetries(offset int64) error {
	hf := hr.file
	renewRetryCtx := hf.newRetryContext()

	for renewRetryCtx.ShouldTry() {
		var err error
		hf.stats.renews++
		hr.currentURL, err = hf.renewURL()
		if err != nil {
			if hf.shouldRetry(err) {
				hf.log("[%9d-%9d] (Connect) retrying %v", offset, offset, err)
				renewRetryCtx.Retry(err)
				continue
			} else {
				hf.log("[%9d-%9d] (Connect) bailing on %v", offset, offset, err)
				return err
			}
		}

		return nil
	}
	return errors.WithMessage(renewRetryCtx.LastError, "httpfile renew")
}

func (hr *httpReader) tryConnect(offset int64) error {
	hf := hr.file

	req, err := http.NewRequest("GET", hf.currentURL, nil)
	if err != nil {
		return err
	}

	byteRange := fmt.Sprintf("bytes=%d-", offset)
	req.Header.Set("Range", byteRange)

	res, err := hf.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == 200 && offset > 0 {
		defer res.Body.Close()
		return errors.WithStack(&ServerError{Host: req.Host, Message: fmt.Sprintf("HTTP Range header not supported"), Code: ServerErrorCodeNoRangeSupport, StatusCode: res.StatusCode})
	}

	if res.StatusCode/100 != 2 {
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			body = []byte("could not read error body")
			err = nil
		}

		if hf.needsRenewal(res, body) {
			return &NeedsRenewalError{url: hf.currentURL}
		}

		return errors.WithStack(&ServerError{Host: req.Host, Message: fmt.Sprintf("HTTP %d: %v", res.StatusCode, string(body)), StatusCode: res.StatusCode})
	}

	hr.Backtracker = backtracker.New(offset, res.Body, maxDiscard)
	hr.body = res.Body
	hr.header = res.Header
	hr.requestURL = res.Request.URL
	hr.statusCode = res.StatusCode
	hr.contentLength = res.ContentLength

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
