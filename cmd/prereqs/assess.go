package prereqs

import (
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/state"
)

type PrereqAssessment struct {
	Done []string
	Todo []string
}

func AssessPrereqs(consumer *state.Consumer, redistRegistry *redist.RedistRegistry, names []string) (*PrereqAssessment, error) {
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

	return pa, nil
}
