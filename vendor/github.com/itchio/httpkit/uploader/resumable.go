package uploader

import (
	"bytes"
	"fmt"
	"io"

	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type resumableUpload struct {
	maxChunkGroup    int
	consumer         *state.Consumer
	progressListener ProgressListenerFunc

	err           error
	splitBuf      *bytes.Buffer
	blocks        chan *rblock
	done          chan struct{}
	cancel        chan struct{}
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
func NewResumableUpload(uploadURL string) ResumableUpload {
	// 64 * 256KiB = 16MiB
	const maxChunkGroup = 64

	id := seed
	seed++
	chunkUploader := &chunkUploader{
		uploadURL:  uploadURL,
		httpClient: timeout.NewClient(resumableConnectTimeout, resumableIdleTimeout),
		id:         id,
	}

	ru := &resumableUpload{
		maxChunkGroup: maxChunkGroup,

		err:           nil,
		splitBuf:      new(bytes.Buffer),
		blocks:        make(chan *rblock, maxChunkGroup),
		done:          make(chan struct{}),
		cancel:        make(chan struct{}),
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
		if ru.err != nil {
			close(ru.cancel)
			return 0, ru.err
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
	if ru.err != nil {
		close(ru.cancel)
		return ru.err
	}

	// flush!
	data := ru.splitBuf.Bytes()
	ru.blocks <- &rblock{
		data: append([]byte{}, data...),
	}
	close(ru.blocks)

	// wait for work() to be done
	<-ru.done

	// return any errors
	if ru.err != nil {
		// no need to bother cancelling anymore, work() has returned
		return ru.err
	}
	return nil
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
	annotatedBlocks := make(chan *rblock)
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
			case <-ru.cancel:
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
			case <-ru.cancel:
				// nevermind, stop everything
				return
			case block := <-annotatedBlocks:
				if block == nil {
					// done receiving blocks!
					break aggregate
				}

				_, err := sendBuf.Write(block.data)
				if err != nil {
					ru.err = errors.WithStack(err)
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
			case <-ru.cancel:
				// nevermind, stop everything
				return
			case block := <-annotatedBlocks:
				if block == nil {
					// done receiving blocks!
					break aggregate
				}

				_, err := sendBuf.Write(block.data)
				if err != nil {
					ru.err = errors.WithStack(err)
					return
				}
				chunkGroupSize++

				if block.last {
					// done receiving blocks
					break aggregate
				}
			default:
				// no more blocks available right now, that's ok
				// let's just send what we got
				break maximize
			}
		}

		// send what we have so far
		ru.debugf("Uploading %d chunks", chunkGroupSize)
		err := ru.chunkUploader.put(sendBuf.Bytes(), false)
		if err != nil {
			ru.err = errors.WithStack(err)
			return
		}
	}

	// send the last block
	ru.debugf("Uploading last %d chunks", chunkGroupSize)
	err := ru.chunkUploader.put(sendBuf.Bytes(), true)
	if err != nil {
		ru.err = errors.WithStack(err)
		return
	}
}

func (ru *resumableUpload) debugf(msg string, args ...interface{}) {
	if ru.consumer != nil {
		fmsg := fmt.Sprintf(msg, args...)
		ru.consumer.Debugf("[ru-%d] %s", ru.id, fmsg)
	}
}
