package prereqs

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/redist"
	"github.com/itchio/ox"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type PrereqsContext struct {
	RequestContext *butlerd.RequestContext
	APIKey         string
	Runtime        *ox.Runtime
	Consumer       *state.Consumer
	PrereqsDir     string
	Force          bool

	library  Library
	registry *redist.RedistRegistry
}

func (pc *PrereqsContext) GetLibrary() (Library, error) {
	if pc.library == nil {
		library, err := NewLibrary(pc.RequestContext, pc.APIKey)
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

		consumer.Infof("Fetching prereqs registry...")
		registry := &redist.RedistRegistry{}

		needFetch := false
		wantFetch := false

		if pc.PrereqsDir == "" {
			return nil, errors.Errorf("PrereqsDir cannot be empty")
		}

		err := os.MkdirAll(pc.PrereqsDir, 0755)
		if err != nil {
			return nil, err
		}

		cachedRegistryPath := filepath.Join(pc.PrereqsDir, "info.json")
		stats, err := os.Stat(cachedRegistryPath)
		if err != nil {
			needFetch = true
		} else {
			sinceLastFetch := time.Since(stats.ModTime())
			if sinceLastFetch > 24*time.Hour {
				consumer.Infof("It's been %s since we fetched the redist registry, let's do it now", sinceLastFetch)
				wantFetch = true
			}
		}
		if pc.Force {
			needFetch = true
		}

		if needFetch || wantFetch {
			err := func() error {
				src, err := eos.Open("https://broth.itch.ovh/itch-redists/info/LATEST/unpacked", option.WithConsumer(pc.Consumer))
				if err != nil {
					return errors.Wrap(err, "opening remote registry file")
				}
				defer src.Close()

				dst, err := os.Create(cachedRegistryPath)
				if err != nil {
					return errors.WithMessage(err, "creating local registry cache")
				}
				defer dst.Close()

				_, err = io.Copy(dst, src)
				if err != nil {
					return errors.WithMessage(err, "downloading registry")
				}
				return nil
			}()
			if err != nil {
				if needFetch {
					return nil, errors.Wrap(err, "fetching prereqs registry")
				} else {
					consumer.Warnf("while fetching prereqs registry: %v", err)
				}
			}
		}

		registryBytes, err := ioutil.ReadFile(cachedRegistryPath)
		if err != nil {
			return nil, errors.WithMessage(err, "while reading registry from disk")
		}

		err = json.Unmarshal(registryBytes, registry)
		if err != nil {
			return nil, errors.WithMessage(err, "decoding redist registry")
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
