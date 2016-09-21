package blockpool

import (
	"fmt"
	"log"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
)

type StoreOp struct {
	loc  BlockLocation
	data []byte
	errs chan error
}

type FanOutSink struct {
	sinks  []Sink
	stores chan StoreOp

	closed bool
	errs   chan error
}

var _ Sink = (*FanOutSink)(nil)

func NewFanOutSink(sinks []Sink) *FanOutSink {
	stores := make(chan StoreOp)

	return &FanOutSink{
		sinks:  sinks,
		stores: stores,
	}
}

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

func (fos *FanOutSink) Store(loc BlockLocation, data []byte) error {
	if fos.closed {
		return errors.Wrap(fmt.Errorf("writing to closed FanOutSink"), 1)
	}

	fos.stores <- StoreOp{
		loc:  loc,
		data: append([]byte{}, data...),
	}

	return nil
}

func (fos *FanOutSink) GetContainer() *tlc.Container {
	return fos.sinks[0].GetContainer()
}
