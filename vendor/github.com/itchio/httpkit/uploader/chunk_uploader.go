package uploader

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type ProgressListenerFunc func(count int64)

type chunkUploader struct {
	// constructor
	uploadURL  string
	httpClient *http.Client
	id         int

	// set later
	progressListener ProgressListenerFunc
	consumer         *state.Consumer

	// internal
	offset int64
	total  int64
}

func (cu *chunkUploader) put(buf []byte, last bool) error {
	retryCtx := cu.newRetryContext()

	for retryCtx.ShouldTry() {
		err := cu.tryPut(buf, last)
		if err != nil {
			if ne, ok := err.(*netError); ok {
				retryCtx.Retry(ne)
				continue
			} else if re, ok := err.(*retryError); ok {
				cu.offset += re.committedBytes
				buf = buf[re.committedBytes:]
				retryCtx.Retry(errors.Errorf("Having troubles uploading some blocks"))
				continue
			} else {
				return errors.WithStack(err)
			}
		} else {
			cu.offset += int64(len(buf))
			return nil
		}
	}

	return fmt.Errorf("Too many errors, stopping upload")
}

func (cu *chunkUploader) tryPut(buf []byte, last bool) error {
	buflen := int64(len(buf))
	if !last && buflen%gcsChunkSize != 0 {
		err := fmt.Errorf("internal error: trying to upload non-last buffer of %d bytes (not a multiple of chunk size %d)",
			buflen, gcsChunkSize)
		return errors.WithStack(err)
	}

	body := bytes.NewReader(buf)
	countingReader := counter.NewReaderCallback(func(count int64) {
		if cu.progressListener != nil {
			cu.progressListener(cu.offset + count)
		}
	}, body)

	req, err := http.NewRequest("PUT", cu.uploadURL, countingReader)
	if err != nil {
		// does not include HTTP errors, more like golang API usage errors
		return errors.WithStack(err)
	}

	start := cu.offset
	end := start + buflen - 1
	contentRange := fmt.Sprintf("bytes %d-%d/*", cu.offset, end)

	if last {
		// send total size
		totalSize := cu.offset + buflen
		contentRange = fmt.Sprintf("bytes %d-%d/%d", cu.offset, end, totalSize)
	}

	req.Header.Set("content-range", contentRange)
	req.ContentLength = buflen
	if last {
		cu.debugf("→ Uploading %d-%d (final slice)", start, end)
	} else {
		cu.debugf("→ Uploading %d-%d (more to come)", start, end)
	}

	startTime := time.Now()

	res, err := cu.httpClient.Do(req)
	if err != nil {
		cu.debugf("while uploading %d-%d: \n%s", start, end, err.Error())
		return &netError{err, GcsUnknown}
	}

	callDuration := time.Since(startTime)
	cu.debugf("← %s (in %s)", res.Status, callDuration)

	status := interpretGcsStatusCode(res.StatusCode)
	if status == GcsUploadComplete && last {
		cu.debugf("✓ %s upload complete!", humanize.IBytes(uint64(cu.offset+buflen)))
		return nil
	}

	if status == GcsNeedQuery {
		cu.debugf("  → Need to query upload status (because of HTTP %s)", res.Status)
		statusRes, err := cu.queryStatus()
		if err != nil {
			// this happens after we retry the query a few times
			return err
		}

		if statusRes.StatusCode == 308 {
			cu.debugf("  ← Got upload status, trying to resume")
			res = statusRes
			status = GcsResume
		} else {
			status = interpretGcsStatusCode(statusRes.StatusCode)
			err = fmt.Errorf("expected upload status, got HTTP %s (%s) instead", statusRes.Status, status)
			cu.debugf(err.Error())
			return err
		}
	}

	if status == GcsResume {
		expectedOffset := cu.offset + buflen
		rangeHeader := res.Header.Get("Range")
		if rangeHeader == "" {
			cu.debugf("X Commit failed (null range), retrying")
			return &retryError{committedBytes: 0}
		}

		committedRange, err := parseRangeHeader(rangeHeader)
		if err != nil {
			return err
		}

		if committedRange.start != 0 {
			return fmt.Errorf("upload failed: beginning not committed somehow (committed range: %s)", committedRange)
		}

		committedBytes := committedRange.end - cu.offset
		perSec := humanize.IBytes(uint64(float64(committedBytes) / callDuration.Seconds()))

		if committedRange.end == expectedOffset {
			cu.debugf("✓ Commit succeeded (%d blocks stored @ %s / s)", buflen/gcsChunkSize, perSec)
			return nil
		}

		if committedBytes < 0 {
			return fmt.Errorf("upload failed: committed negative bytes somehow (committed range: %s, expectedOffset: %d)", committedRange, expectedOffset)
		}

		if committedBytes > 0 {
			cu.debugf("✓ Commit partially succeeded (%d / %d byte, %d blocks stored @ %s / s)", committedBytes, buflen, committedBytes/gcsChunkSize, perSec)
			return &retryError{committedBytes}
		}

		cu.debugf("X Commit failed (retrying %d blocks)", buflen/gcsChunkSize)
		return &retryError{committedBytes}
	}

	return fmt.Errorf("got HTTP %d (%s)", res.StatusCode, status)
}

func (cu *chunkUploader) queryStatus() (*http.Response, error) {
	retryCtx := cu.newRetryContext()
	for retryCtx.ShouldTry() {
		res, err := cu.tryQueryStatus()
		if err != nil {
			cu.debugf("while querying status of upload: %s", err.Error())
			retryCtx.Retry(err)
			continue
		}

		return res, nil
	}

	return nil, fmt.Errorf("gave up on trying to get upload status")
}

func (cu *chunkUploader) tryQueryStatus() (*http.Response, error) {
	req, err := http.NewRequest("PUT", cu.uploadURL, nil)
	if err != nil {
		// does not include HTTP errors, more like golang API usage errors
		return nil, errors.WithStack(err)
	}

	// for resumable uploads of unknown size, the length is unknown,
	// see https://github.com/itchio/butler/issues/71#issuecomment-242938495
	req.Header.Set("content-range", "bytes */*")

	res, err := cu.httpClient.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	status := interpretGcsStatusCode(res.StatusCode)
	if status == GcsResume {
		// got what we wanted (Range header, etc.)
		return res, nil
	}

	return nil, fmt.Errorf("while querying status, got HTTP %s (status %s)", res.Status, status)
}

func (cu *chunkUploader) debugf(msg string, args ...interface{}) {
	if cu.consumer != nil {
		fmsg := fmt.Sprintf(msg, args...)
		cu.consumer.Debugf("[cu-%d] %s", cu.id, fmsg)
	}
}

func (cu *chunkUploader) newRetryContext() *retrycontext.Context {
	return retrycontext.New(retrycontext.Settings{
		MaxTries: resumableMaxRetries,
		Consumer: cu.consumer,
	})
}
