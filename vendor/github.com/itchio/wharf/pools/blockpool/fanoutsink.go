package blockpool

import (
	"fmt"

	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

type storeOp struct {
	loc  BlockLocation
	data []byte
	errs chan error
}

// A FanOutSink distributes Store calls to a number of underlying stores.
type FanOutSink struct {
	sinks  []Sink
	stores chan storeOp

	closed    bool
	cancelled chan struct{}
	errs      chan error
}

var _ Sink = (*FanOutSink)(nil)

// Clone returns a copy of the sink
func (fos *FanOutSink) Clone() Sink {
	sinkClones := make([]Sink, len(fos.sinks))
	for i, sink := range fos.sinks {
		sinkClones[i] = sink.Clone()
	}

	return &FanOutSink{
		sinks:  sinkClones,
		stores: make(chan storeOp),
	}
}

// NewFanOutSink returns a newly initialized FanOutSink
func NewFanOutSink(templateSink Sink, numSinks int) (*FanOutSink, error) {
	if numSinks <= 0 {
		return nil, errors.WithStack(fmt.Errorf("numSinks must > 0, was %d", numSinks))
	}

	stores := make(chan storeOp)
	sinks := make([]Sink, numSinks)

	for i := 0; i < numSinks; i++ {
		sinks[i] = templateSink.Clone()
	}

	fos := &FanOutSink{
		sinks:  sinks,
		stores: stores,
	}
	return fos, nil
}

// Start processing store requests
func (fos *FanOutSink) Start() {
	fos.errs = make(chan error, len(fos.sinks))
	fos.cancelled = make(chan struct{})

	for _, sink := range fos.sinks {
		go func(sink Sink) {
			for {
				select {
				case <-fos.cancelled:
					// stop handling requests!
					fos.errs <- nil
					return
				case store, ok := <-fos.stores:
					if !ok {
						// no more requests to handle
						fos.errs <- nil
						return
					}

					err := sink.Store(store.loc, store.data)
					if err != nil {
						fos.errs <- err
						return
					}
				}
			}
		}(sink)
	}
}

// Close is the only way to retrieve errors from a fan-out sink,
// since individual Store calls will never fail.
func (fos *FanOutSink) Close() error {
	if fos.closed {
		return nil
	}

	fos.closed = true
	close(fos.stores)

	// wait for all workers to finish
	for i := 0; i < len(fos.sinks); i++ {
		err := <-fos.errs
		if err != nil {
			return err
		}
	}

	return nil
}

// Store has slightly different semantics compared to typical sinks -
// it will sometimes immediately return, and always return nil (no error).
// To retrieve errors, one has to call Close explicitly
func (fos *FanOutSink) Store(loc BlockLocation, data []byte) error {
	if fos.closed {
		return errors.WithStack(fmt.Errorf("writing to closed FanOutSink"))
	}

	op := storeOp{
		loc:  loc,
		data: append([]byte{}, data...),
	}

	select {
	case err := <-fos.errs:
		// stop accepting stores
		fos.closed = true

		// cancel all workers
		close(fos.cancelled)

		// immediately return error
		return err
	case fos.stores <- op:
		// we managed to send the store!
	}

	return nil
}

// GetContainer returns the container associated with this fan-in sink
func (fos *FanOutSink) GetContainer() *tlc.Container {
	return fos.sinks[0].GetContainer()
}
