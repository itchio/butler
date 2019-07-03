package bfs

import (
	"os"

	"github.com/itchio/lake/tlc"
)

func Walk(path string) (*tlc.Container, error) {
	return tlc.WalkDir(path, &tlc.WalkOpts{Filter: DotItchFilter()})
}

func DotItchFilter() tlc.FilterFunc {
	return func(fi os.FileInfo) bool {
		// skip directories named ".itch". in WalkDir, this
		// will also skip all its children
		if fi.IsDir() && fi.Name() == ".itch" {
			return false
		}

		// walk everything else
		return true
	}
}

// ContainerPaths returns a list of all paths in a
// container, for all files and symlinks. Folders
// are excluded
func ContainerPaths(container *tlc.Container) []string {
	res := []string{}
	for _, f := range container.Files {
		res = append(res, f.Path)
	}

	for _, s := range container.Symlinks {
		res = append(res, s.Path)
	}
	return res
}

// Return elements in b that aren't in a
func Difference(a []string, b []string) []string {
	// struct{} = 0-sized type, we're using it to
	// use `map` as a set.
	aMap := make(map[string]struct{})
	for _, el := range a {
		aMap[el] = struct{}{}
	}

	var res = []string{}
	for _, el := range b {
		if _, ok := aMap[el]; !ok {
			res = append(res, el)
		}
	}

	return res
}

func SliceToLength(a []string, length int) []string {
	if a == nil {
		return a
	}

	return a[:min(length, len(a))]
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
