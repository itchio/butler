package blockpool

import (
	"time"

	"github.com/itchio/wharf/tlc"
)

/////////////////
// Sink
/////////////////

// A DelayedSink waits a set amount of time before delegating the store to the
// underling sink.
type DelayedSink struct {
	Sink    Sink
	Latency time.Duration
}

var _ Sink = (*DelayedSink)(nil)

// Clone returns a copy of the delayed sink, storing to a copy of the underlying sink
func (ds *DelayedSink) Clone() Sink {
	return &DelayedSink{
		Sink:    ds.Sink.Clone(),
		Latency: ds.Latency,
	}
}

// Store behaves just like the underlying sink, with a delay
func (ds *DelayedSink) Store(loc BlockLocation, data []byte) error {
	time.Sleep(ds.Latency)
	return ds.Sink.Store(loc, data)
}

// GetContainer returns the underlying sink's container
func (ds *DelayedSink) GetContainer() *tlc.Container {
	return ds.Sink.GetContainer()
}

/////////////////
// Source
/////////////////

// A DelayedSource waits a set amount of time before delegating the fetch to the
// underling source.
type DelayedSource struct {
	Source  Source
	Latency time.Duration
}

var _ Source = (*DelayedSource)(nil)

// Clone returns a copy of the delayed sink, storing to a copy of the underlying sink
func (ds *DelayedSource) Clone() Source {
	return &DelayedSource{
		Source:  ds.Source.Clone(),
		Latency: ds.Latency,
	}
}

// Fetch behaves just like the underlying source, with a delay
func (ds *DelayedSource) Fetch(loc BlockLocation, data []byte) (int, error) {
	time.Sleep(ds.Latency)
	return ds.Source.Fetch(loc, data)
}

// GetContainer returns the underlying source's containre
func (ds *DelayedSource) GetContainer() *tlc.Container {
	return ds.Source.GetContainer()
}
