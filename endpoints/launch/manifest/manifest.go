package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/mitchellh/mapstructure"

	"github.com/BurntSushi/toml"

	"github.com/pkg/errors"
)

// TODO: linter

func ListActions(m *butlerd.Manifest, runtime *manager.Runtime) []*butlerd.Action {
	var result []*butlerd.Action

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

func Path(folder string) string {
	return filepath.Join(folder, ".itch.toml")
}

// Read an itch app manifest from a folder
// Returns a nil manifest if there isn't an `.itch.toml` file
// in the folder. Returns an error if there is a file, but it can't
// be read, for example because of permissions errors, invalid TOML
// markup, or invalid manifest structure
func Read(folder string) (*butlerd.Manifest, error) {
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

	manifest := &butlerd.Manifest{}
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

func ExpandPath(a *butlerd.Action, runtime *manager.Runtime, baseFolder string) string {
	if filepath.IsAbs(a.Path) {
		return a.Path
	}

	path := a.Path
	if strings.Contains(path, "{{EXT}}") {
		var ext = ""
		switch runtime.Platform {
		case butlerd.ItchPlatformWindows:
			ext = ".exe"
		case butlerd.ItchPlatformOSX:
			ext = ".app"
		}
		path = strings.Replace(path, "{{EXT}}", ext, 1)
	}

	return filepath.Join(baseFolder, path)
}
