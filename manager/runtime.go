package manager

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
)

type SupportedRuntimes []SupportedRuntime

type RuntimeEnumerator interface {
	// Returns a list of supported runtimes, from most preferred
	// to least preferred
	Enumerate() (SupportedRuntimes, error)
}

type defaultRuntimeEnumerator struct{}

var _ RuntimeEnumerator = (*defaultRuntimeEnumerator)(nil)

func DefaultRuntimeEnumerator() RuntimeEnumerator {
	return &defaultRuntimeEnumerator{}
}

func (dre *defaultRuntimeEnumerator) Enumerate() (SupportedRuntimes, error) {
	rts := SupportedRuntimes{
		SupportedRuntime{
			Runtime: ox.CurrentRuntime(),
		},
	}
	return rts, nil
}

func (sr SupportedRuntimes) IsCompatible(p itchio.Platforms) bool {
	for _, r := range sr {
		if IsCompatible(p, r.Runtime) {
			return true
		}
	}
	return false
}

type singleRuntimeEnumerator struct {
	rt ox.Runtime
}

var _ RuntimeEnumerator = (*singleRuntimeEnumerator)(nil)

func SingleRuntimeEnumerator(rt ox.Runtime) RuntimeEnumerator {
	return &singleRuntimeEnumerator{
		rt: rt,
	}
}

func (sre *singleRuntimeEnumerator) Enumerate() (SupportedRuntimes, error) {
	res := SupportedRuntimes{
		SupportedRuntime{Runtime: sre.rt},
	}
	return res, nil
}
