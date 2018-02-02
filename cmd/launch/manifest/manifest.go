package manifest

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"

	"github.com/BurntSushi/toml"

	"github.com/go-errors/errors"
)

type Manifest struct {
	Actions []*Action
	Prereqs []*Prereq
}

type Action struct {
	Name    string
	Path    string
	Args    []string
	Scope   string
	Sandbox bool

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
	_, err = toml.DecodeReader(f, intermediate)
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

func (a *Action) GetPath(baseFolder string) string {
	if filepath.IsAbs(a.Path) {
		return a.Path
	}
	return filepath.Join(baseFolder, a.Path)
}
