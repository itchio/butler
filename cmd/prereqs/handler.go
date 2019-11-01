package prereqs

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/redist"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

type Handler interface{}

type Params struct {
	RequestContext *butlerd.RequestContext
	APIKey         string
	Host           manager.Host
	Consumer       *state.Consumer
	PrereqsDir     string
	Force          bool
}

type handler struct {
	params Params

	library  Library
	registry *redist.RedistRegistry
}

func NewHandler(params Params) (Handler, error) {
	err := validation.ValidateStruct(&params,
		validation.Field(&params.RequestContext, validation.Required),
	)
	if err != nil {
		return nil, err
	}

	return &prereqsContext{
		params: params,
	}, nil
}

func (pc *prereqsContext) runtime() ox.Runtime {
	return pc.params.Host.Runtime
}

func (pc *prereqsContext) GetLibrary() (Library, error) {
	if pc.library == nil {
		library, err := NewLibrary(pc.params.RequestContext, pc.runtime(), pc.params.APIKey)
		if err != nil {
			return nil, errors.Wrap(err, "opening prereqs library")
		}

		pc.library = library
	}
	return pc.library, nil
}

func (pc *prereqsContext) GetRegistry() (*redist.RedistRegistry, error) {
	if pc.registry == nil {
		beforeFetch := time.Now()

		consumer := pc.params.Consumer

		consumer.Infof("Fetching prereqs registry...")
		registry := &redist.RedistRegistry{}

		needFetch := false
		wantFetch := false

		if pc.params.PrereqsDir == "" {
			return nil, errors.Errorf("PrereqsDir cannot be empty")
		}

		err := os.MkdirAll(pc.params.PrereqsDir, 0o755)
		if err != nil {
			return nil, err
		}

		cachedRegistryPath := filepath.Join(pc.params.PrereqsDir, "info.json")
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
		if pc.params.Force {
			needFetch = true
		}

		if needFetch || wantFetch {
			err := func() error {
				src, err := eos.Open("https://broth.itch.ovh/itch-redists/info/LATEST/unpacked", option.WithConsumer(pc.params.Consumer))
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

func (pc *prereqsContext) GetEntry(name string) (*redist.RedistEntry, error) {
	r, err := pc.GetRegistry()
	if err != nil {
		return nil, errors.Wrap(err, "opening prereqs registry")
	}

	return r.Entries[name], nil
}

func (pc *prereqsContext) GetEntryDir(name string) string {
	return filepath.Join(pc.params.PrereqsDir, name)
}
