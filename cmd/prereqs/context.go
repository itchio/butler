package prereqs

import (
	"encoding/json"
	"path/filepath"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type PrereqsContext struct {
	RequestContext *butlerd.RequestContext
	Credentials    *butlerd.GameCredentials
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
			return nil, errors.Wrap(err, "opening prereqs library")
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
			return nil, errors.Wrap(err, "opening prereqs library")
		}

		consumer.Infof("Fetching prereqs registry...")
		registry := &redist.RedistRegistry{}

		err = func() error {
			registryURL, err := library.GetURL("info", "unpacked")
			if err != nil {
				return errors.Wrap(err, "getting URL for redist registry")
			}

			f, err := eos.Open(registryURL)
			if err != nil {
				return errors.Wrap(err, "opening remote registry file")
			}
			defer f.Close()

			dec := json.NewDecoder(f)
			err = dec.Decode(registry)
			if err != nil {
				return errors.Wrap(err, "decoding redist registry")
			}

			return nil
		}()
		if err != nil {
			return nil, errors.Wrap(err, "fetching prereqs registry")
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
		return nil, errors.Wrap(err, "opening prereqs registry")
	}

	return r.Entries[name], nil
}

func (pc *PrereqsContext) GetEntryDir(name string) string {
	return filepath.Join(pc.PrereqsDir, name)
}
