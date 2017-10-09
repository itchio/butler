package archive

import (
	"path/filepath"
)

func CleanFileName(fileName string) string {
	// input path may be with `\` or `/` separator
	// this normalizes to `/`
	nativeFileName := filepath.ToSlash(fileName)

	// clean returns the shortest possible path,
	// resolving `..`, double separators etc.
	cleanedNative := filepath.Clean(nativeFileName)

	// clean's output uses the native separator, so
	// `\` on windows.
	// output path should always use `/` separator
	cleanedSlash := filepath.ToSlash(cleanedNative)

	return cleanedSlash
}
