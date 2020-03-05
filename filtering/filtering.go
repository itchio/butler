package filtering

import (
	"path/filepath"

	"github.com/itchio/lake/tlc"
)

var CustomIgnorePatterns = []string{}

// FilterPaths filters out known bad folder/files
// which butler should just ignore
var FilterPaths tlc.FilterFunc = func(name string) tlc.FilterResult {
	if tlc.PresetFilter(name) == tlc.FilterIgnore {
		return tlc.FilterIgnore
	}

	for _, pattern := range CustomIgnorePatterns {
		match, _ := filepath.Match(pattern, name)
		if match {
			return tlc.FilterIgnore
		}
	}

	return tlc.FilterKeep
}

