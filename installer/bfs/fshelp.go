package bfs

import (
	"os"
)

func Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func Mkdir(path string) error {
	return os.MkdirAll(path, 0o755)
}
