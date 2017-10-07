package unarchiver

import (
	"encoding/json"
	"os/exec"

	"github.com/go-errors/errors"
)

type ListResult struct {
	FormatVersion int64    `json:"lsarFormatVersion"`
	Contents      []*Entry `json:"lsarContents"`
	Encoding      string   `json:"lsarEncoding"`
	Confidence    int64    `json:"lsarConfidence"`
	FormatName    string   `json:"lsarFormatName"`
}

// Entry contains the fields we care about most often
type Entry struct {
	XADFileName string
	XADFileSize int64
}

type FullEntry struct {
	Entry

	XADIndex      int64
	XADPosixGroup int64
	XADPosixUser  int64
	// examples: "None",
	XADCompressionName string
	XADDataLength      int64
	ZipLocalDate       int64

	/////////////////////////////////////////
	// date format: "2016-09-27 02:15:12 +0200"
	/////////////////////////////////////////
	XADLastModificationDate string
	XADLastAccessDate       string

	/////////////////////////////////////////
	// zip-specific entries
	/////////////////////////////////////////
	ZipOS             int64
	ZipOSName         string
	ZipExtractVersion int64
	ZipFlags          int64
	ZipCRC32          int64
	ZipFileAttributes int64
}

func List(path string) (*ListResult, error) {
	cmd := exec.Command("lsar", "--json", path)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := ListResult{}
	err = json.NewDecoder(stdout).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}

func (lr *ListResult) TotalSize() int64 {
	var total int64
	for _, entry := range lr.Contents {
		total += entry.XADFileSize
	}
	return total
}
