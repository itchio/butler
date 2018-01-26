package filtering

import (
	"fmt"
	"os"
	"path/filepath"
)

var IgnoredPaths = []string{
	".git",
	".hg",
	".svn",
	".DS_Store",
	"__MACOSX",
	"._*",
	"Thumbs.db",
	".itch",
}

// FilterPaths filters out known bad folder/files
// which butler should just ignore
func FilterPaths(fileInfo os.FileInfo) bool {
	name := fileInfo.Name()
	for _, pattern := range IgnoredPaths {
		match, err := filepath.Match(pattern, name)
		if err != nil {
			panicMsg := fmt.Sprintf("Malformed ignore pattern '%s': %s", pattern, err.Error())
			panic(panicMsg)
		}
		if match {
			return false
		}
	}

	return true
}
