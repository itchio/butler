package operate

import "github.com/itchio/wharf/state"

type SlicedProgress struct {
	consumer *state.Consumer
	start    float64
	end      float64
}

func (sp SlicedProgress) Progress(p float64) {
	adjustedP := sp.start + p/1.0*(sp.end-sp.start)
	sp.consumer.Progress(adjustedP)
}
