package prereqs

import (
	"encoding/json"
	"fmt"
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

type Handler interface {
	HasInstallMarker(name string) bool
	MarkInstalled(name string) error

	GetEntry(name string) (*redist.RedistEntry, error)
	GetRegistry() (*redist.RedistRegistry, error)

	FilterPrereqs(names []string) ([]string, error)
	AssessPrereqs(names []string) (*PrereqAssessment, error)
	FetchPrereqs(tsc *TaskStateConsumer, names []string) error
	BuildPlan(names []string) (*PrereqPlan, error)
	InstallPrereqs(tsc *TaskStateConsumer, plan *PrereqPlan) error
}

var _ Handler = (*handler)(nil)

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

	return &handler{
		params: params,
	}, nil
}

func (h *handler) prereqsDir() string {
	return h.params.PrereqsDir
}

func (h *handler) rc() *butlerd.RequestContext {
	return h.params.RequestContext
}

func (h *handler) consumer() *state.Consumer {
	return h.params.Consumer
}

func (h *handler) host() manager.Host {
	return h.params.Host
}

func (h *handler) runtime() ox.Runtime {
	return h.host().Runtime
}

func (h *handler) platform() ox.Platform {
	return h.runtime().Platform
}

func (h *handler) GetLibrary() (Library, error) {
	if h.library == nil {
		library, err := NewLibrary(h.rc(), h.runtime(), h.params.APIKey)
		if err != nil {
			return nil, errors.Wrap(err, "opening prereqs library")
		}

		h.library = library
	}
	return h.library, nil
}

func (h *handler) GetRegistry() (*redist.RedistRegistry, error) {
	if h.registry == nil {
		beforeFetch := time.Now()

		consumer := h.consumer()

		consumer.Infof("Fetching prereqs registry...")
		registry := &redist.RedistRegistry{}

		needFetch := false
		wantFetch := false

		if h.params.PrereqsDir == "" {
			return nil, errors.Errorf("PrereqsDir cannot be empty")
		}

		err := os.MkdirAll(h.params.PrereqsDir, 0o755)
		if err != nil {
			return nil, err
		}

		cachedRegistryPath := filepath.Join(h.params.PrereqsDir, "info.json")
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
		if h.params.Force {
			needFetch = true
		}

		if needFetch || wantFetch {
			err := func() error {
				src, err := eos.Open("https://broth.itch.ovh/itch-redists/info/LATEST/unpacked", option.WithConsumer(h.consumer()))
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
				}
				consumer.Warnf("while fetching prereqs registry: %v", err)
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

		h.registry = registry
	}

	return h.registry, nil
}

func (h *handler) GetEntry(name string) (*redist.RedistEntry, error) {
	r, err := h.GetRegistry()
	if err != nil {
		return nil, errors.Wrap(err, "opening prereqs registry")
	}

	return r.Entries[name], nil
}

func (h *handler) GetEntryDir(name string) string {
	if !h.runtime().Equals(ox.CurrentRuntime()) {
		prefix := fmt.Sprintf("%s-%s", h.runtime().OS(), h.runtime().Arch())
		return filepath.Join(h.prereqsDir(), prefix, name)
	} else {
		return filepath.Join(h.prereqsDir(), name)
	}
}
