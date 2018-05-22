package tlc

import (
	"fmt"
	"os"

	"github.com/itchio/httpkit/progress"
)

func (f *File) ToString() string {
	return fmt.Sprintf("%s %10s %s", os.FileMode(f.Mode), progress.FormatBytes(f.Size), f.Path)
}

func (f *Dir) ToString() string {
	return fmt.Sprintf("%s %10s %s/", os.FileMode(f.Mode), "-", f.Path)
}

func (f *Symlink) ToString() string {
	return fmt.Sprintf("%s %10s %s -> %s", os.FileMode(f.Mode), "-", f.Path, f.Dest)
}

type WriteLine func(line string)

func (container *Container) Print(output WriteLine) {
	for _, f := range container.Dirs {
		output(f.ToString())
	}
	for _, f := range container.Symlinks {
		output(f.ToString())
	}
	for _, f := range container.Files {
		output(f.ToString())
	}
}
