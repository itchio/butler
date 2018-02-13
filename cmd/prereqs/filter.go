package prereqs

import (
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/state"
)

func FilterPrereqs(consumer *state.Consumer, redistRegistry *redist.RedistRegistry, names []string) ([]string, error) {
	var result []string
	for _, name := range names {
		entry, ok := redistRegistry.Entries[name]
		if !ok {
			consumer.Warnf("Prereq (%s) not found in registry, skipping...", name)
			continue
		}

		if RedistHasPlatform(entry, manager.CurrentRuntime().Platform) {
			result = append(result, name)
		}
	}
	return result, nil
}

func RedistHasPlatform(redist *redist.RedistEntry, platform manager.ItchPlatform) bool {
	for _, p := range redist.Platforms {
		if p == string(platform) {
			return true
		}
	}
	return false
}
