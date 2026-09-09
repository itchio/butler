package steam

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// SyncEntry is one app to sync to one itch.io project. It is the record
// the CLI reads from a sync config file and the one the app stores for a
// connection, so it carries no secrets and nothing machine specific.
type SyncEntry struct {
	App    uint32 `toml:"app" json:"app"`
	Target string `toml:"target" json:"target"`
	// Branch defaults to "public".
	Branch string   `toml:"branch,omitempty" json:"branch,omitempty"`
	Hidden bool     `toml:"hidden,omitempty" json:"hidden,omitempty"`
	Skip   []uint32 `toml:"skip,omitempty" json:"skip,omitempty"`
	// Map sends a depot to a channel by hand, keyed by depot id. TOML
	// keys are strings, so the ids are strings here; see Mapping.
	Map map[string]string `toml:"map,omitempty" json:"map,omitempty"`
	// CacheDir keeps depots between syncs. A relative path is relative to
	// the config file, and an entry without one falls back to the config
	// wide cache dir, see SyncConfig.Resolve.
	CacheDir string `toml:"cache_dir,omitempty" json:"cacheDir,omitempty"`
}

func (e SyncEntry) BranchOrDefault() string {
	if e.Branch == "" {
		return "public"
	}
	return e.Branch
}

func (e SyncEntry) Mapping() (map[uint32]string, error) {
	out := map[uint32]string{}
	for k, v := range e.Map {
		id, err := strconv.ParseUint(k, 10, 32)
		if err != nil || v == "" {
			return nil, fmt.Errorf("bad map entry %q = %q, want DEPOTID = \"channel\"", k, v)
		}
		out[uint32(id)] = v
	}
	return out, nil
}

func (e SyncEntry) SkipSet() map[uint32]bool {
	out := map[uint32]bool{}
	for _, id := range e.Skip {
		out[id] = true
	}
	return out
}

func (e SyncEntry) PlanOptions(password string) (PlanOptions, error) {
	m, err := e.Mapping()
	if err != nil {
		return PlanOptions{}, err
	}
	return PlanOptions{
		AppID:    e.App,
		Branch:   e.BranchOrDefault(),
		Password: password,
		Target:   e.Target,
		Map:      m,
		Skip:     e.SkipSet(),
	}, nil
}

func (e SyncEntry) Validate() error {
	if e.App == 0 {
		return errors.New("app is required")
	}
	if e.Target == "" {
		return errors.New("target is required")
	}
	if _, err := e.Mapping(); err != nil {
		return err
	}
	return nil
}

// SyncConfig is a sync config file.
type SyncConfig struct {
	// CacheDir is the cache dir for every entry that does not set its
	// own. Each app stages in its own subdirectory named by app id.
	CacheDir string      `toml:"cache_dir,omitempty" json:"cacheDir,omitempty"`
	Sync     []SyncEntry `toml:"sync" json:"sync"`

	// Entries keep paths as written; Resolve makes them absolute.
	dir string
}

func LoadSyncConfig(path string) (*SyncConfig, error) {
	var c SyncConfig
	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, errors.Wrapf(err, "reading %s", path)
	}
	c.dir = filepath.Dir(path)
	seen := map[uint32]bool{}
	for i := range c.Sync {
		e := &c.Sync[i]
		if err := e.Validate(); err != nil {
			return nil, errors.Wrapf(err, "%s: sync entry %d", path, i+1)
		}
		if seen[e.App] {
			return nil, errors.Errorf("%s: app %d appears more than once", path, e.App)
		}
		seen[e.App] = true
	}
	return &c, nil
}

// Resolve fills in the cache dir for an entry: its own if set, otherwise
// <config cache dir>/<app id>. A relative path either way is relative to
// the config file.
func (c *SyncConfig) Resolve(e SyncEntry) SyncEntry {
	if e.CacheDir == "" && c.CacheDir != "" {
		e.CacheDir = filepath.Join(c.CacheDir, strconv.FormatUint(uint64(e.App), 10))
	}
	if e.CacheDir != "" && !filepath.IsAbs(e.CacheDir) && c.dir != "" {
		e.CacheDir = filepath.Join(c.dir, e.CacheDir)
	}
	return e
}

func (c *SyncConfig) Find(app uint32) *SyncEntry {
	for i := range c.Sync {
		if c.Sync[i].App == app {
			return &c.Sync[i]
		}
	}
	return nil
}
