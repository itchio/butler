package tlc

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

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

var (
	ErrUnrecognizedContainer = errors.New("Unrecognized container: should either be a directory, or a .zip archive")
)

// A FilterFunc allows ignoring certain files or directories when walking the filesystem
// When a directory is ignored by a FilterFunc, all its children are, too!
type FilterFunc func(fileInfo os.FileInfo) bool

// DefaultFilter is a passthrough that filters out no files at all
var DefaultFilter FilterFunc = func(fileInfo os.FileInfo) bool {
	return true
}

type WalkOpts struct {
	// Filter decides which files to exclude from the walk
	Filter FilterFunc

	// Dereference walks symlinks as if they were their targets
	Dereference bool
}

// WalkAny tries to retrieve container information on containerPath. It supports:
// the empty container (/dev/null), local directories, zip archives, or single files
func WalkAny(containerPath string, opts *WalkOpts) (*Container, error) {
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
		// local directory case
		return WalkDir(containerPath, opts)
	}

	// zip archive case
	if strings.HasSuffix(strings.ToLower(stat.Name()), ".zip") {
		zr, err := zip.NewReader(file, stat.Size())
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		return WalkZip(zr, opts)
	}

	// single file case
	return WalkSingle(file)
}

// WalkSingle returns a container with a single file
func WalkSingle(file eos.File) (*Container, error) {
	stats, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !stats.Mode().IsRegular() {
		return nil, errors.Wrap(fmt.Errorf("%s: not a regular file, can only WalkSingle regular files", stats.Name()), 0)
	}

	container := &Container{
		Files: []*File{
			&File{
				Mode:   uint32(stats.Mode()),
				Size:   int64(stats.Size()),
				Offset: 0,
				Path:   filepath.Base(stats.Name()),
			},
		},
		Size: stats.Size(),
	}
	return container, nil
}

// WalkDir retrieves information on all files, directories, and symlinks in a directory
func WalkDir(basePathIn string, opts *WalkOpts) (*Container, error) {
	filter := opts.Filter

	if filter == nil {
		filter = DefaultFilter
	}

	var Dirs []*Dir
	var Symlinks []*Symlink
	var Files []*File

	currentlyWalking := make(map[string]bool)

	TotalOffset := int64(0)

	var makeEntryCallback func(BasePath string, LocationPath string) filepath.WalkFunc

	makeEntryCallback = func(BasePath string, LocationPath string) filepath.WalkFunc {
		return func(FullPath string, fileInfo os.FileInfo, err error) error {
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

			Path = filepath.Join(LocationPath, Path)

			Path = filepath.ToSlash(Path)
			if Path == "." {
				// Don't store a single folder named "."
				return nil
			}

			// os.Walk does not follow symlinks, so we must do it
			// manually if Dereference is set
			if opts.Dereference && fileInfo.Mode()&os.ModeSymlink > 0 {
				fileInfo, err = os.Stat(FullPath)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if fileInfo.Mode().IsDir() {
					Dest, err := os.Readlink(FullPath)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					var JoinedDest string
					if filepath.IsAbs(Dest) {
						JoinedDest = Dest
					} else {
						JoinedDest = filepath.Join(filepath.Dir(FullPath), Dest)
					}

					CleanDest := filepath.Clean(JoinedDest)

					if currentlyWalking[CleanDest] {
						err := fmt.Errorf("symlinks recurse onto %s, cowardly refusing to walk infinite container", CleanDest)
						return errors.Wrap(err, 0)
					}

					currentlyWalking[CleanDest] = true
					err = filepath.Walk(CleanDest, makeEntryCallback(CleanDest, Path))
					delete(currentlyWalking, CleanDest)
					if err != nil {
						return errors.Wrap(err, 0)
					}
				}
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
	}

	if basePathIn == NullPath {
		// empty container is fine - /dev/null is legal even on Win32 where it doesn't exist
	} else {
		basePathIn, err := filepath.Abs(basePathIn)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		fi, err := os.Lstat(basePathIn)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		if !fi.IsDir() {
			return nil, errors.Wrap(fmt.Errorf("can't walk non-directory %s", basePathIn), 1)
		}

		currentlyWalking[basePathIn] = true
		err = filepath.Walk(basePathIn, makeEntryCallback(basePathIn, "."))
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
	}

	container := &Container{Size: TotalOffset, Dirs: Dirs, Symlinks: Symlinks, Files: Files}
	return container, nil
}

// WalkZip walks all file in a zip archive and returns a container
func WalkZip(zr *zip.Reader, opts *WalkOpts) (*Container, error) {
	filter := opts.Filter

	if filter == nil {
		// default filter is a passthrough
		filter = func(fileInfo os.FileInfo) bool {
			return true
		}
	}

	if opts.Dereference {
		return nil, errors.New("Dereference is not supporting when walking a zip")
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

// IsSingleFile returns true if the container contains
// exactly one files, and no directories or symlinks.
func (container *Container) IsSingleFile() bool {
	if len(container.Files) == 1 && len(container.Dirs) == 0 && len(container.Symlinks) == 0 {
		return true
	}
	return false
}
