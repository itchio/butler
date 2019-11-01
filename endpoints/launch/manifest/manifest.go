package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/ox"
	"github.com/mitchellh/mapstructure"

	"github.com/BurntSushi/toml"

	"github.com/pkg/errors"
)

// A Manifest describes prerequisites (dependencies) and actions that
// can be taken while launching a game.
type Manifest struct {
	// Actions are a list of options to give the user when launching a game.
	Actions Actions `json:"actions"`

	// Prereqs describe libraries or frameworks that must be installed
	// prior to launching a game
	Prereqs []Prereq `json:"prereqs,omitempty"`
}

type Actions []Action

// An Action is a choice for the user to pick when launching a game.
//
// see https://itch.io/docs/itch/integrating/manifest.html
type Action struct {
	// human-readable or standard name
	Name string `json:"name"`

	// file path (relative to manifest or absolute), URL, etc.
	Path string `json:"path"`

	// icon name (see static/fonts/icomoon/demo.html, don't include `icon-` prefix)
	Icon string `json:"icon,omitempty"`

	// command-line arguments
	Args []string `json:"args,omitempty"`

	// sandbox opt-in
	Sandbox bool `json:"sandbox,omitempty"`

	// requested API scope
	Scope string `json:"scope,omitempty"`

	// don't redirect stdout/stderr, open in new console window
	Console bool `json:"console,omitempty"`

	// platform to restrict this action to
	Platform ox.Platform `json:"platform,omitempty"`

	// localized action name
	Locales map[string]*ActionLocale `json:"locales,omitempty"`
}

func (a Action) RunsOn(platform ox.Platform) bool {
	if a.Platform == "" {
		return true
	}
	return a.Platform == platform
}

type Prereq struct {
	// A prerequisite to be installed, see <https://itch.io/docs/itch/integrating/prereqs/> for the full list.
	Name string `json:"name"`
}

type ActionLocale struct {
	// A localized action name
	Name string `json:"name"`
}

func (actions Actions) FilterByPlatform(platform ox.Platform) []Action {
	var result Actions

	for _, a := range actions {
		if a.RunsOn(platform) {
			result = append(result, a)
		}
	}

	return result
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
		return nil, errors.WithStack(err)
	}

	defer f.Close()

	intermediate := make(map[string]interface{})
	_, err = toml.DecodeReader(f, &intermediate)
	if err != nil {
		// invalid TOML
		return nil, errors.WithStack(err)
	}

	manifest := &Manifest{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: manifest,
	})
	if err != nil {
		// internal error
		return nil, errors.WithStack(err)
	}

	err = decoder.Decode(intermediate)
	if err != nil {
		// invalid manifest structure
		return nil, errors.WithStack(err)
	}

	return manifest, nil
}

func (a Action) ExpandPath(platform ox.Platform, baseFolder string) string {
	if filepath.IsAbs(a.Path) {
		return a.Path
	}

	path := a.Path
	if strings.Contains(path, "{{EXT}}") {
		var ext = ""
		switch platform {
		case ox.PlatformWindows:
			ext = ".exe"
		case ox.PlatformOSX:
			ext = ".app"
		}
		path = strings.Replace(path, "{{EXT}}", ext, 1)
	}

	return filepath.Join(baseFolder, path)
}
