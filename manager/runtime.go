package manager

import (
	"os/exec"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/itchio/ox"
)

type SupportedRuntimes []SupportedRuntime

type RuntimeEnumerator interface {
	// Returns a list of supported runtimes, from most preferred
	// to least preferred
	Enumerate(consumer *state.Consumer) (SupportedRuntimes, error)
}

type defaultRuntimeEnumerator struct{}

var _ RuntimeEnumerator = (*defaultRuntimeEnumerator)(nil)

func DefaultRuntimeEnumerator() RuntimeEnumerator {
	return &defaultRuntimeEnumerator{}
}

func (dre *defaultRuntimeEnumerator) Enumerate(consumer *state.Consumer) (SupportedRuntimes, error) {
	native := SupportedRuntime{
		Runtime: ox.CurrentRuntime(),
	}
	consumer.Debugf("Native platform: %v", native)

	rts := SupportedRuntimes{
		native,
	}

	if native.Runtime.Platform != ox.PlatformWindows {
		consumer.Debugf("Looking for wine...")

		// determine if wine is installed
		winePath, err := exec.LookPath("wine")
		if err == nil {
			consumer.Debugf("Found wine at: (%s)", winePath)
			rts = append(rts, SupportedRuntime{
				Runtime: ox.Runtime{
					Platform: ox.PlatformWindows,
					Is64:     false, // 32-bit windows supports both
				},
				Wrapper: &Wrapper{
					WrapperBinary: winePath,
				},
			})
		} else {
			consumer.Debugf("While looking for wine: %+v", err)
		}
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

func (sre *singleRuntimeEnumerator) Enumerate(consumer *state.Consumer) (SupportedRuntimes, error) {
	res := SupportedRuntimes{
		SupportedRuntime{Runtime: sre.rt},
	}
	return res, nil
}
