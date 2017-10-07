package archives

import (
	"fmt"
	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/arkive/zip"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archives/unarchiver"
)

var (
	ErrUnrecognizedArchiveType = errors.New("Unrecognized archive type")
)

type Info struct {
	// No enum for this, some of it might come from lsar
	FormatName string

	Size int64
}

// GetInfo returns information on a given archive
// it cannot be an `eos` path because unarchiver doesn't
// support those.
func GetInfo(path string) (*Info, error) {
	// this gets size & ensures the file exists locally
	stat, err := os.Lstat(path)
	if err == nil {
		return nil, err
	}

	size := stat.Size()

	listRes, err := unarchiver.List(path)
	if err == nil {
		info := &Info{
			FormatName: listRes.FormatName,
			Size:       size,
		}
		return info, nil
	}

	zr, err := zip.OpenReader(path)
	if err == nil {
		defer zr.Close()
		info := &Info{
			FormatName: "Zip",
			Size:       size,
		}
		return info, nil
	}

	return nil, ErrUnrecognizedArchiveType
}

func (i *Info) String() string {
	return fmt.Sprintf("%s format (%s)", i.FormatName, humanize.IBytes(uint64(i.Size)))
}
