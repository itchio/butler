package tlc

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
)

// Prepare creates all directories, files, and symlinks.
// It also applies the proper permissions if the files already exist
func (c *Container) Prepare(basePath string) error {
	for _, dirEntry := range c.Dirs {
		err := c.prepareDir(basePath, dirEntry)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	for _, fileEntry := range c.Files {
		err := c.prepareFile(basePath, fileEntry)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	for _, link := range c.Symlinks {
		err := c.prepareSymlink(basePath, link)
		if err != nil {
			return errors.Wrap(err, 1)
		}
	}

	return nil
}

func (c *Container) prepareDir(basePath string, dirEntry *Dir) error {
	fullPath := filepath.Join(basePath, dirEntry.Path)
	err := os.MkdirAll(fullPath, os.FileMode(dirEntry.Mode))
	if err != nil {
		return errors.Wrap(err, 1)
	}
	err = os.Chmod(fullPath, os.FileMode(dirEntry.Mode))
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func (c *Container) prepareFile(basePath string, fileEntry *File) error {
	fullPath := filepath.Join(basePath, fileEntry.Path)
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_TRUNC, os.FileMode(fileEntry.Mode))
	if err != nil {
		return errors.Wrap(err, 1)
	}
	err = file.Close()
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = os.Truncate(fullPath, fileEntry.Size)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	// if file already exists, opening with O_TRUNC doesn't change its permissions
	err = os.Chmod(fullPath, os.FileMode(fileEntry.Mode))
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func (c *Container) prepareSymlink(basePath string, link *Symlink) error {
	fullPath := filepath.Join(basePath, link.Path)
	err := os.RemoveAll(fullPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = os.Symlink(link.Dest, fullPath)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}
