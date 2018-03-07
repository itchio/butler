package prereqs

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type PrereqsContext struct {
	RequestContext *buse.RequestContext
	Credentials    *buse.GameCredentials
	Runtime        *manager.Runtime
	Consumer       *state.Consumer
	PrereqsDir     string

	library  Library
	registry *redist.RedistRegistry
}

func (pc *PrereqsContext) GetLibrary() (Library, error) {
	if pc.library == nil {
		library, err := NewLibrary(pc.Credentials)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		pc.library = library
	}
	return pc.library, nil
}

func (pc *PrereqsContext) GetRegistry() (*redist.RedistRegistry, error) {
	if pc.registry == nil {
		beforeFetch := time.Now()

		consumer := pc.Consumer

		library, err := pc.GetLibrary()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		consumer.Infof("Fetching prereqs registry...")
		registry := &redist.RedistRegistry{}

		err = func() error {
			registryURL, err := library.GetURL("info", "unpacked")
			if err != nil {
				return errors.Wrap(err, 0)
			}

			f, err := eos.Open(registryURL)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			defer f.Close()

			dec := json.NewDecoder(f)
			err = dec.Decode(registry)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		registryFetchDuration := time.Since(beforeFetch)
		consumer.Infof("✓ Fetched %d entries in %s", len(registry.Entries), registryFetchDuration)

		pc.registry = registry
	}

	return pc.registry, nil
}

func (pc *PrereqsContext) GetEntry(name string) (*redist.RedistEntry, error) {
	r, err := pc.GetRegistry()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return r.Entries[name], nil
}

func (pc *PrereqsContext) GetEntryDir(name string) string {
	return filepath.Join(pc.PrereqsDir, name)
}
