package prereqs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/state"
)

type PrereqAssessment struct {
	Done []string
	Todo []string
}

func AssessPrereqs(consumer *state.Consumer, redistRegistry *redist.RedistRegistry, prereqsDir string, names []string) (*PrereqAssessment, error) {
	pa := &PrereqAssessment{}

	for _, name := range names {
		entry, ok := redistRegistry.Entries[name]
		if !ok {
			consumer.Warnf("Prereq (%s) not found in registry, skipping...", name)
			continue
		}

		hasRegistry := false

		for _, registryKey := range entry.RegistryKeys {
			if RegistryKeyExists(consumer, registryKey) {
				hasRegistry = true
				break
			}
		}

		if hasRegistry {
			// then it's already installed, cool!
			pa.Done = append(pa.Done, name)
			continue
		}

		pa.Todo = append(pa.Todo, name)
	}

	for _, name := range pa.Done {
		err := MarkInstalled(prereqsDir, name)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		continue
	}

	return pa, nil
}

func MarkerPath(prereqsDir string, name string) string {
	return filepath.Join(prereqsDir, name, ".installed")
}

func IsInstalled(prereqsDir string, name string) bool {
	path := MarkerPath(prereqsDir, name)
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return false
}

func MarkInstalled(prereqsDir string, name string) error {
	if IsInstalled(prereqsDir, name) {
		// don't mark again
		return nil
	}

	contents := fmt.Sprintf("Installed on %s", time.Now())
	path := MarkerPath(prereqsDir, name)
	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0755))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = ioutil.WriteFile(path, []byte(contents), os.FileMode(0644))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
