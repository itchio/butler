package prereqs

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/manager"
	"github.com/itchio/butler/redist"
)

func (pc *PrereqsContext) FilterPrereqs(names []string) ([]string, error) {
	consumer := pc.Consumer

	var result []string
	for _, name := range names {
		entry, err := pc.GetEntry(name)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping...", name)
			continue
		}

		if !RedistHasPlatform(entry, pc.Runtime.Platform) {
			consumer.Warnf("Prereq (%s) is not relevant on (%s), skipping...", name, pc.Runtime.Platform)
			continue
		}
		result = append(result, name)
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
