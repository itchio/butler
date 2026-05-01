package walkutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/lake/tlc"
)

// ResolveSingleZipDir returns the path to an inner .zip if `path` is a
// directory whose only non-filtered entry is a regular .zip file, otherwise
// returns `path` unchanged. When it rewrites, it logs via comm.Opf so the
// user can see why butler is operating on the zip rather than the directory.
func ResolveSingleZipDir(path string, filter tlc.FilterFunc) string {
	fi, err := os.Stat(path)
	if err != nil || !fi.IsDir() {
		return path
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return path
	}

	var keep []os.DirEntry
	for _, e := range entries {
		if filter(e.Name()) == tlc.FilterIgnore {
			continue
		}
		keep = append(keep, e)
	}
	if len(keep) != 1 {
		return path
	}

	only := keep[0]
	if !strings.HasSuffix(strings.ToLower(only.Name()), ".zip") {
		return path
	}

	info, err := only.Info()
	if err != nil || !info.Mode().IsRegular() {
		return path
	}

	inner := filepath.Join(path, only.Name())
	comm.Opf("(%s) contains a single .zip file, treating %s as the container", path, only.Name())
	return inner
}
