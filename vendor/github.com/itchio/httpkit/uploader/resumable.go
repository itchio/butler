package uploader

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type resumableUpload struct {
	maxChunkGroup    int
	consumer         *state.Consumer
	progressListener ProgressListenerFunc

	closed        bool
	err           error
	errMu         sync.RWMutex
	pushedErr     chan struct{}
	splitBuf      *bytes.Buffer
	blocks        chan *rblock
	done          chan struct{}
	chunkUploader *chunkUploader
	id            int
}

// ResumableUpload represents a resumable upload session
// to google cloud storage.
type ResumableUpload interface {
	io.WriteCloser
	SetConsumer(consumer *state.Consumer)
	SetProgressListener(progressListener ProgressListenerFunc)
}

type rblock struct {
	data []byte
	last bool
}

const rblockSize = 256 * 1024

var seed = 0

var _ ResumableUpload = (*resumableUpload)(nil)

// NewResumableUpload starts a new resumable upload session
// targeting the specified Google Cloud Storage uploadURL.
func NewResumableUpload(uploadURL string, opts ...Option) ResumableUpload {
	s := defaultSettings()
	for _, o := range opts {
		o.Apply(s)
	}

	id := seed
	seed++
	chunkUploader := &chunkUploader{
		uploadURL:  uploadURL,
		httpClient: timeout.NewClient(resumableConnectTimeout, resumableIdleTimeout),
		id:         id,
	}

	ru := &resumableUpload{
		maxChunkGroup: s.MaxChunkGroup,

		err:           nil,
		pushedErr:     make(chan struct{}, 0),
		splitBuf:      new(bytes.Buffer),
		blocks:        make(chan *rblock),
		done:          make(chan struct{}, 0),
		chunkUploader: chunkUploader,
		id:            id,
	}
	ru.splitBuf.Grow(rblockSize)

	go ru.work()

	return ru
}

// Write implements io.Writer.
func (ru *resumableUpload) Write(buf []byte) (int, error) {
	sb := ru.splitBuf

	written := 0
	for written < len(buf) {
		if err := ru.checkError(); err != nil {
			return 0, err
		}
		if ru.closed {
			return 0, nil
		}

		availRead := len(buf) - written
		availWrite := sb.Cap() - sb.Len()

		if availWrite == 0 {
			// flush!
			data := sb.Bytes()
			ru.blocks <- &rblock{
				data: append([]byte{}, data...),
			}
			sb.Reset()
			availWrite = sb.Cap()
		}

		copySize := availRead
		if copySize > availWrite {
			copySize = availWrite
		}

		// buffer!
		sb.Write(buf[written : written+copySize])
		written += copySize
	}

	return written, nil
}

// Close implements io.Closer.
func (ru *resumableUpload) Close() error {
	if err := ru.checkError(); err != nil {
		return err
	}

	if ru.closed {
		return nil
	}
	ru.closed = true

	// flush!
	data := ru.splitBuf.Bytes()
	ru.blocks <- &rblock{
		data: append([]byte{}, data...),
	}
	close(ru.blocks)

	// wait for work() to be done
	select {
	case <-ru.done: // muffin
	case <-ru.pushedErr: // muffin
	}

	// return any errors
	return ru.checkError()
}

func (ru *resumableUpload) SetConsumer(consumer *state.Consumer) {
	ru.consumer = consumer
	ru.chunkUploader.consumer = consumer
}

func (ru *resumableUpload) SetProgressListener(progressListener ProgressListenerFunc) {
	ru.chunkUploader.progressListener = progressListener
}

//===========================================
// internal functions
//===========================================

func (ru *resumableUpload) work() {
	defer close(ru.done)

	sendBuf := new(bytes.Buffer)
	sendBuf.Grow(ru.maxChunkGroup * rblockSize)
	var chunkGroupSize int

	// same as ru.blocks, but `.last` is set properly, no matter
	// what the size is
	annotatedBlocks := make(chan *rblock, ru.maxChunkGroup)
	go func() {
		var lastBlock *rblock
		defer close(annotatedBlocks)

	annotate:
		for {
			select {
			case b := <-ru.blocks:
				if b == nil {
					// ru.blocks closed!
					break annotate
				}

				// queue block
				if lastBlock != nil {
					annotatedBlocks <- lastBlock
				}
				lastBlock = b
			case <-ru.pushedErr:
				// stop everything
				return
			}
		}

		if lastBlock != nil {
			lastBlock.last = true
			annotatedBlocks <- lastBlock
		}
	}()

aggregate:
	for {
		sendBuf.Reset()
		chunkGroupSize = 0

		{
			// do a block receive for the first vlock
			select {
			case <-ru.pushedErr:
				// nevermind, stop everything
				return
			case block := <-annotatedBlocks:
				if block == nil {
					// done receiving blocks!
					break aggregate
				}

				_, err := sendBuf.Write(block.data)
				if err != nil {
					ru.pushError(errors.WithStack(err))
					return
				}
				chunkGroupSize++

				if block.last {
					// done receiving blocks
					break aggregate
				}
			}
		}

		// see if we can't gather any more blocks
	maximize:
		for chunkGroupSize < ru.maxChunkGroup {
			select {
			case <-ru.pushedErr:
				// nevermind, stop everything
				return
			case block := <-annotatedBlocks:
				if block == nil {
					// done receiving blocks!
					break aggregate
				}

				_, err := sendBuf.Write(block.data)
				if err != nil {
					ru.pushError(errors.WithStack(err))
					return
				}
				chunkGroupSize++

				if block.last {
					// done receiving blocks
					break aggregate
				}
			case <-time.After(100 * time.Millisecond):
				// no more blocks available right now, that's ok
				// let's just send what we got
				break maximize
			}
		}

		// send what we have so far
		ru.debugf("Uploading %d chunks", chunkGroupSize)
		err := ru.chunkUploader.put(sendBuf.Bytes(), false)
		if err != nil {
			ru.pushError(errors.WithStack(err))
			return
		}
	}

	// send the last block
	ru.debugf("Uploading last %d chunks", chunkGroupSize)
	err := ru.chunkUploader.put(sendBuf.Bytes(), true)
	if err != nil {
		ru.pushError(errors.WithStack(err))
		return
	}
}

func (ru *resumableUpload) debugf(msg string, args ...interface{}) {
	if ru.consumer != nil {
		fmsg := fmt.Sprintf(msg, args...)
		ru.consumer.Debugf("[ru-%d] %s", ru.id, fmsg)
	}
}

func (ru *resumableUpload) checkError() error {
	ru.errMu.RLock()
	err := ru.err
	ru.errMu.RUnlock()
	return err
}

func (ru *resumableUpload) pushError(err error) {
	ru.errMu.Lock()
	if ru.err != nil {
		ru.errMu.Unlock()
		return
	}
	ru.err = err
	close(ru.pushedErr)
	ru.errMu.Unlock()
}
