package tlc

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/itchio/arkive/zip"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
)

const (
	// ModeMask is or'd with files being diffed
	ModeMask = 0644

	// NullPath can be specified instead of a directory to yield an empty container
	NullPath = "/dev/null"
)

// A FilterFunc allows ignoring certain files or directories when walking the filesystem
// When a directory is ignored by a FilterFunc, all its children are, too!
type FilterFunc func(fileInfo os.FileInfo) bool

// DefaultFilter is a passthrough that filters out no files at all
var DefaultFilter FilterFunc = func(fileInfo os.FileInfo) bool {
	return true
}

// WalkAny tries to retrieve container information on containerPath. It supports:
// the empty container (/dev/null), local directories, zip archives
func WalkAny(containerPath string, filter FilterFunc) (*Container, error) {
	// empty container case
	if containerPath == NullPath {
		return &Container{}, nil
	}

	file, err := eos.Open(containerPath)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	if stat.IsDir() {
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		// local directory case
		return WalkDir(containerPath, filter)
	}

	// zip archive case
	zr, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return WalkZip(zr, filter)
}

// WalkDir retrieves information on all files, directories, and symlinks in a directory
func WalkDir(BasePath string, filter FilterFunc) (*Container, error) {
	if filter == nil {
		filter = DefaultFilter
	}

	var Dirs []*Dir
	var Symlinks []*Symlink
	var Files []*File

	TotalOffset := int64(0)

	onEntry := func(FullPath string, fileInfo os.FileInfo, err error) error {
		// we shouldn't encounter any error crawling the repo
		if err != nil {
			if os.IsPermission(err) {
				// ...except permission errors, those are fine
				log.Printf("Permission error: %s\n", err.Error())
			} else {
				return errors.Wrap(err, 1)
			}
		}

		Path, err := filepath.Rel(BasePath, FullPath)
		if err != nil {
			return errors.Wrap(err, 1)
		}

		Path = filepath.ToSlash(Path)
		if Path == "." {
			// Don't store a single folder named "."
			return nil
		}

		// don't end up with files we (the patcher) can't modify
		Mode := fileInfo.Mode() | ModeMask

		if !filter(fileInfo) {
			if Mode.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if Mode.IsDir() {
			Dirs = append(Dirs, &Dir{Path: Path, Mode: uint32(Mode)})
		} else if Mode.IsRegular() {
			Size := fileInfo.Size()
			Offset := TotalOffset
			OffsetEnd := Offset + Size

			Files = append(Files, &File{Path: Path, Mode: uint32(Mode), Size: Size, Offset: Offset})
			TotalOffset = OffsetEnd
		} else if Mode&os.ModeSymlink > 0 {
			Dest, err := os.Readlink(FullPath)
			if err != nil {
				return errors.Wrap(err, 1)
			}

			Dest = filepath.ToSlash(Dest)
			Symlinks = append(Symlinks, &Symlink{Path: Path, Mode: uint32(Mode), Dest: Dest})
		}

		return nil
	}

	if BasePath == NullPath {
		// empty container is fine - /dev/null is legal even on Win32 where it doesn't exist
	} else {
		fi, err := os.Lstat(BasePath)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		if !fi.IsDir() {
			return nil, errors.Wrap(fmt.Errorf("can't walk non-directory %s", BasePath), 1)
		}

		err = filepath.Walk(BasePath, onEntry)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	container := &Container{Size: TotalOffset, Dirs: Dirs, Symlinks: Symlinks, Files: Files}
	return container, nil
}

// WalkZip walks all file in a zip archive and returns a container
func WalkZip(zr *zip.Reader, filter FilterFunc) (*Container, error) {
	if filter == nil {
		// default filter is a passthrough
		filter = func(fileInfo os.FileInfo) bool {
			return true
		}
	}

	var Dirs []*Dir
	var Symlinks []*Symlink
	var Files []*File

	dirMap := make(map[string]os.FileMode)

	TotalOffset := int64(0)

	for _, file := range zr.File {
		fileName := filepath.ToSlash(filepath.Clean(filepath.ToSlash(file.Name)))

		// don't trust zip files to have directory entries for
		// all directories. it's a miracle anything works.
		dir := path.Dir(fileName)
		if dir != "" && dir != "." && dirMap[dir] == 0 {
			dirMap[dir] = os.FileMode(0755)
		}

		info := file.FileInfo()
		mode := file.Mode() | ModeMask

		if info.IsDir() {
			dirMap[fileName] = mode
		} else if mode&os.ModeSymlink > 0 {
			var linkname []byte

			err := func() error {
				reader, err := file.Open()
				if err != nil {
					return errors.Wrap(err, 1)
				}
				defer reader.Close()

				linkname, err = ioutil.ReadAll(reader)
				if err != nil {
					return errors.Wrap(err, 1)
				}
				return nil
			}()

			if err != nil {
				return nil, errors.Wrap(err, 1)
			}

			Symlinks = append(Symlinks, &Symlink{
				Path: fileName,
				Dest: string(linkname),
				Mode: uint32(mode),
			})
		} else {
			Size := int64(file.UncompressedSize64)

			Files = append(Files, &File{
				Path:   fileName,
				Mode:   uint32(mode),
				Size:   Size,
				Offset: TotalOffset,
			})

			TotalOffset += Size
		}
	}

	for dirPath, dirMode := range dirMap {
		Dirs = append(Dirs, &Dir{
			Path: dirPath,
			Mode: uint32(dirMode),
		})
	}

	container := &Container{
		Size:     TotalOffset,
		Dirs:     Dirs,
		Symlinks: Symlinks,
		Files:    Files,
	}
	return container, nil
}

// Stats return a human-readable summary of the contents of a container
func (container *Container) Stats() string {
	return fmt.Sprintf("%d files, %d dirs, %d symlinks",
		len(container.Files), len(container.Dirs), len(container.Symlinks))
}
