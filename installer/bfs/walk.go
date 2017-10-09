package bfs

import (
	"os"
	"path/filepath"

	"github.com/itchio/wharf/tlc"
)

func Walk(path string) (*tlc.Container, error) {
	return tlc.WalkDir(path, DotItchFilter(path))
}

func DotItchFilter(basePath string) tlc.FilterFunc {
	return func(fi os.FileInfo) bool {
		absolutePath := fi.Name()

		relativePath, err := filepath.Rel(basePath, absolutePath)
		if err != nil {
			panic(err)
		}

		dir, _ := filepath.Split(relativePath)
		isInItch := filepath.Clean(dir) == ".itch"

		// walk everything that isn't in Itch
		return !isInItch
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
