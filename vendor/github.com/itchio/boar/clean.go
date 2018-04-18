package boar

import (
	"path"
	"strings"
)

func CleanFileName(fileName string) string {
	// input path may be with `\` or `/` separator
	// this normalizes to `/`
	// we can't use `filepath.ToSlash` because it depends
	// on the OS path separator, but we want cross-platform
	// results
	slashName := strings.Replace(fileName, `\`, `/`, -1)

	// clean returns the shortest possible path,
	// resolving `..`, double separators etc.
	return path.Clean(slashName)
}
