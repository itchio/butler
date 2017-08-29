package pwr

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
)

// A LockMap is an array of channels, corresponding to file indices
// of a container. If set, a healer must attempt to receive from the
// corresponding channel before starting to heal a file. Users of healers
// should generally pass an array of fresh channels and close them once
// the file becomes available for healing.
type LockMap []chan interface{}

func NewLockMap(container *tlc.Container) LockMap {
	lockMap := make([]chan interface{}, len(container.Files))
	for i := range lockMap {
		lockMap[i] = make(chan interface{})
	}
	return lockMap
}

// A Healer consumes wounds and tries to repair them by creating
// directories, symbolic links, and writing the correct data into files.
type Healer interface {
	WoundsConsumer

	SetNumWorkers(numWorkers int)
	SetConsumer(consumer *state.Consumer)
	SetLockMap(lockmap LockMap)
	TotalHealed() int64
}

func NewHealer(spec string, target string) (Healer, error) {
	return NewHealerEx(spec, target, true)
}

// NewHealer takes a spec of the form "type,url", and a target folder
// and returns a healer that knows how to repair target from spec.
func NewHealerEx(spec string, target string, lazy bool) (Healer, error) {
	makeHealer := func() (Healer, error) {
		tokens := strings.SplitN(spec, ",", 2)
		if len(tokens) != 2 {
			return nil, fmt.Errorf("Invalid healer spec: expected 'type,url' but got '%s'", spec)
		}

		healerType := tokens[0]
		healerURL := tokens[1]

		switch healerType {
		case "archive":
			file, err := eos.Open(healerURL)
			if err != nil {
				return nil, errors.Wrap(err, 1)
			}

			ah := &ArchiveHealer{
				File:   file,
				Target: target,
			}
			return ah, nil
		case "manifest":
			return nil, fmt.Errorf("Manifest healer: stub")
		}

		return nil, fmt.Errorf("Unknown healer type %s", healerType)
	}

	if lazy {
		return &LazyHealer{
			makeHealer: makeHealer,
		}, nil
	}
	return makeHealer()
}
