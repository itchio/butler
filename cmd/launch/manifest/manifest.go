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
	Actions []*Action `json:"actions"`
	Prereqs []*Prereq `json:"prereqs"`
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
	Name string `json:"name"`

	// file path (relative to manifest or absolute), URL, etc.
	Path string `json:"path"`

	// icon name (see static/fonts/icomoon/demo.html, don't include `icon-` prefix)
	Icon string `json:"icon"`

	// command-line arguments
	Args []string `json:"args"`

	// sandbox opt-in
	Sandbox bool `json:"sandbox"`

	// requested API scope
	Scope string `json:"scope"`

	// don't redirect stdout/stderr, open in new console window
	Console bool `json:"console"`

	// platform to restrict this action too
	Platform manager.ItchPlatform `json:"platform"`

	// localized action name
	Locales map[string]*ActionLocale `json:"locales"`
}

type Prereq struct {
	Name string `json:"name"`
}

type ActionLocale struct {
	Name string `json:"name"`
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
		ext := ""
		switch runtime.Platform {
		case manager.ItchPlatformWindows:
			ext = ".exe"
		case manager.ItchPlatformOSX:
			ext = ".app"
		}
		path = strings.Replace(path, "{{EXT}}", ext, 1)
	}

	return filepath.Join(baseFolder, path)
}
