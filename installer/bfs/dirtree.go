package bfs

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type dirnode map[string]dirnode

type DirTree struct {
	basePath string
	root     dirnode
}

func NewDirTree(basePath string) *DirTree {
	return &DirTree{
		basePath: basePath,
		root:     make(dirnode),
	}
}

// EnsureParents makes sure that all the parent directories of a given
// path exist
func (dt *DirTree) EnsureParents(filePath string) error {
	dirPath := path.Dir(filePath)

	if dt.hasPath(filePath) {
		// cool, we have it!
		return nil
	}

	// ok, let's make it!
	err := os.MkdirAll(filepath.Join(dt.basePath, dirPath), 0o755)
	if err != nil {
		// mkdirall will return `ENOTDIR` if one of the elements is not a directory
		return errors.Wrap(err, "dirtree ensuring parents for file")
	}

	dt.commitPath(filePath)

	return nil
}

func (dt *DirTree) CommitFiles(filePaths []string) {
	for _, filePath := range filePaths {
		dt.commitPath(path.Dir(filePath))
	}
}

type WalkFunc func(name string, node dirnode)

// ListRelativeDirs returns a list of directories in this
// tree, relative to the tree's base path, depth first
func (dt *DirTree) ListRelativeDirs() []string {
	res := []string{}

	var walk WalkFunc
	walk = func(name string, node dirnode) {
		for childName, childNode := range node {
			walk(path.Join(name, childName), childNode)
		}

		res = append(res, name)
	}
	walk(".", dt.root)

	return res
}

func (dt *DirTree) split(dirPath string) []string {
	return strings.Split(path.Clean(dirPath), "/")
}

func (dt *DirTree) hasPath(dirPath string) bool {
	tokens := dt.split(dirPath)
	node := dt.root
	for _, token := range tokens {
		if nextNode, ok := node[token]; ok {
			node = nextNode
		} else {
			return false
		}
	}
	return true
}

func (dt *DirTree) commitPath(dirPath string) {
	if dirPath == "." {
		return
	}

	tokens := dt.split(dirPath)
	node := dt.root
	for _, token := range tokens {
		if nextNode, ok := node[token]; ok {
			node = nextNode
		} else {
			newNode := make(dirnode)
			node[token] = newNode
			node = newNode
		}
	}
}
