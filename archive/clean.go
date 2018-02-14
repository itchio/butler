package archive

import (
	"path/filepath"
)

func CleanFileName(fileName string) string {
	// clean returns the shortest possible path,
	// resolving `..`, double separators etc.
	cleanedNative := filepath.Clean(fileName)

	// clean's output uses the native separator, so
	// `\` on windows.
	// output path should always use `/` separator
	cleanedSlash := filepath.ToSlash(cleanedNative)

	return cleanedSlash
}
