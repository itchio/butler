package blockpool

import (
	"fmt"
	"log"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
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

	closed bool
	errs   chan error
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
		return nil, errors.Wrap(fmt.Errorf("numSinks must > 0, was %d", numSinks), 1)
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
	fos.errs = make(chan error)

	log.Printf("Starting FanOutSink with %d sinks", len(fos.sinks))

	for _, sink := range fos.sinks {
		go func(sink Sink) {
			for store := range fos.stores {
				err := sink.Store(store.loc, store.data)
				if err != nil {
					fos.errs <- err
					return
				}
			}
			fos.errs <- nil
		}(sink)
	}
}

// Close is the only way to retrieve errors from a fan-out sink,
// since individual Store calls will never fail.
func (fos *FanOutSink) Close() error {
	fos.closed = true
	close(fos.stores)

	var rErr error

	for i := 0; i < len(fos.sinks); i++ {
		err := <-fos.errs
		if err != nil {
			rErr = err
		}
	}

	return rErr
}

// Store has slightly different semantics compared to typical sinks -
// it will sometimes immediately return, and always return nil (no error).
// To retrieve errors, one has to call Close explicitly
func (fos *FanOutSink) Store(loc BlockLocation, data []byte) error {
	if fos.closed {
		return errors.Wrap(fmt.Errorf("writing to closed FanOutSink"), 1)
	}

	fos.stores <- storeOp{
		loc:  loc,
		data: append([]byte{}, data...),
	}

	return nil
}

// GetContainer returns the container associated with this fan-in sink
func (fos *FanOutSink) GetContainer() *tlc.Container {
	return fos.sinks[0].GetContainer()
}
