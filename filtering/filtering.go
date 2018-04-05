package filtering

import (
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
		match, _ := filepath.Match(pattern, name)
		if match {
			return false
		}
	}

	return true
}
