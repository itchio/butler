package blockpool

import (
	"time"

	"github.com/itchio/wharf/tlc"
)

/////////////////
// Sink
/////////////////

type DelayedSink struct {
	Sink    Sink
	Latency time.Duration
}

func (ds *DelayedSink) Store(loc BlockLocation, data []byte) error {
	time.Sleep(ds.Latency)
	return ds.Sink.Store(loc, data)
}

func (ds *DelayedSink) GetContainer() *tlc.Container {
	return ds.Sink.GetContainer()
}

/////////////////
// Source
/////////////////

type DelayedSource struct {
	Source  Source
	Latency time.Duration
}

func (ds *DelayedSource) Fetch(loc BlockLocation) ([]byte, error) {
	time.Sleep(ds.Latency)
	return ds.Source.Fetch(loc)
}

func (ds *DelayedSource) GetContainer() *tlc.Container {
	return ds.Source.GetContainer()
}
