package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/manager"
	"github.com/mitchellh/mapstructure"

	"github.com/BurntSushi/toml"

	"github.com/go-errors/errors"
)

// TODO: linter

type Manifest struct {
	Actions []*Action
	Prereqs []*Prereq
}

func (m *Manifest) ListActions(runtime *manager.Runtime) []*Action {
	var result []*Action

	for _, a := range m.Actions {
		if a.Platform == "" {
			// universal
			result = append(result, a)
		} else if a.Platform == runtime.Platform {
			// just the right platform for us!
			result = append(result, a)
		}
		// otherwise, skip it
	}

	return result
}

// see https://itch.io/docs/itch/integrating/manifest.html
type Action struct {
	// human-readable or standard name
	Name string

	// file path (relative to manifest or absolute), URL, etc.
	Path string

	// icon name (see static/fonts/icomoon/demo.html, don't include `icon-` prefix)
	Icon string

	// command-line arguments
	Args []string

	// sandbox opt-in
	Sandbox bool

	// requested API scope
	Scope string

	// don't redirect stdout/stderr, open in new console window
	Console bool

	// platform to restrict this action too
	Platform manager.ItchPlatform

	// localized action name
	Locales map[string]*ActionLocale
}

type Prereq struct {
	Name string
}

type ActionLocale struct {
	Name string
}

func Path(folder string) string {
	return filepath.Join(folder, ".itch.toml")
}

// Read an itch app manifest from a folder
// Returns a nil manifest if there isn't an `.itch.toml` file
// in the folder. Returns an error if there is a file, but it can't
// be read, for example because of permissions errors, invalid TOML
// markup, or invalid manifest structure
func Read(folder string) (*Manifest, error) {
	manifestPath := Path(folder)
	f, err := os.Open(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			// no manifest!
			return nil, nil
		}
		return nil, errors.Wrap(err, 0)
	}

	defer f.Close()

	intermediate := make(map[string]interface{})
	_, err = toml.DecodeReader(f, &intermediate)
	if err != nil {
		// invalid TOML
		return nil, errors.Wrap(err, 0)
	}

	manifest := &Manifest{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: manifest,
	})
	if err != nil {
		// internal error
		return nil, errors.Wrap(err, 0)
	}

	err = decoder.Decode(intermediate)
	if err != nil {
		// invalid manifest structure
		return nil, errors.Wrap(err, 0)
	}

	return manifest, nil
}

func (a *Action) ExpandPath(runtime *manager.Runtime, baseFolder string) string {
	if filepath.IsAbs(a.Path) {
		return a.Path
	}

	path := a.Path
	if strings.Contains(path, "{{EXT}}") {
		var ext = ""
		switch runtime.Platform {
		case manager.ItchPlatformWindows:
			ext = ".exe"
		}
		path = strings.Replace(path, "{{EXT}}", ext, 1)
	}

	return filepath.Join(baseFolder, path)
}
