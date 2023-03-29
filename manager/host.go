package manager

import (
	"fmt"
	"os/exec"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

type Hosts []Host

type HostEnumerator interface {
	// Returns a list of supported hosts, from most preferred
	// to least preferred
	Enumerate(consumer *state.Consumer) (Hosts, error)
}

type defaultHostEnumerator struct{}

var _ HostEnumerator = (*defaultHostEnumerator)(nil)

func DefaultHostEnumerator() HostEnumerator {
	return &defaultHostEnumerator{}
}

func (h Host) Validate() error {
	if h.Runtime.Platform == "" {
		return errors.Errorf("invalid host (empty platform)")
	}
	return nil
}

func NativeHost() Host {
	return Host{
		Runtime: ox.CurrentRuntime(),
	}
}

func (dre *defaultHostEnumerator) Enumerate(consumer *state.Consumer) (Hosts, error) {
	native := NativeHost()
	consumer.Debugf("Native platform: %v", native)

	rts := Hosts{
		native,
	}

	if native.Runtime.Platform != ox.PlatformWindows {
		consumer.Debugf("Looking for wine...")

		// determine if wine is installed, fallback to flatpak wine
		winePath, wineErr := exec.LookPath("wine")
		pakPath, pakErr := exec.LookPath("org.winehq.Wine")

		foundWinePath := nil

		if wineErr =! nil {
			foundWinePath := winePath
		} else if pakErr =! nil {
			foundWinePath := pakPath
		}

		if foundWinePath =! nil {
			consumer.Debugf("Found wine at: (%s)", winePath)
			rts = append(rts, Host{
				Runtime: ox.Runtime{
					Platform: ox.PlatformWindows,
					Is64:     false, // 32-bit windows supports both
				},
				Wrapper: &Wrapper{
					WrapperBinary:      foundWinePath,
					NeedRelativeTarget: true,
				},
			})
		} else {
			consumer.Debugf("While looking for wine:")
			consumer.Debugf("	%+v", wineErr)
			consumer.Debugf("	%+v", pakErr)
		}
	}

	return rts, nil
}

func (h Host) String() string {
	res := h.Runtime.String()
	if h.RemoteLaunchName != "" {
		res += fmt.Sprintf(" (remoteLaunchName=%s)", h.RemoteLaunchName)
	} else if h.Wrapper != nil {
		res += fmt.Sprintf(" (wrapper=%s)", h.Wrapper.WrapperBinary)
	} else {
		res += " (native)"
	}
	return res
}

func (h Hosts) IsCompatible(p itchio.Platforms) bool {
	for _, r := range h {
		if IsCompatible(p, r.Runtime) {
			return true
		}
	}
	return false
}

func (h Hosts) Platforms() []ox.Platform {
	var platforms []ox.Platform

	for _, r := range h {
		// don't add platforms twice
		for _, p := range platforms {
			if p == r.Runtime.Platform {
				continue
			}
		}
		platforms = append(platforms, r.Runtime.Platform)
	}

	return platforms
}

type singleHostEnumerator struct {
	rt ox.Runtime
}

var _ HostEnumerator = (*singleHostEnumerator)(nil)

func SingleHostEnumerator(rt ox.Runtime) HostEnumerator {
	return &singleHostEnumerator{
		rt: rt,
	}
}

func (sre *singleHostEnumerator) Enumerate(consumer *state.Consumer) (Hosts, error) {
	res := Hosts{
		Host{Runtime: sre.rt},
	}
	return res, nil
}
