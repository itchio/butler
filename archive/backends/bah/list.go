package bah

import (
	"archive/zip"

	"github.com/itchio/butler/archive"
)

func (h *Handler) List(params *archive.ListParams) (archive.ListResult, error) {
	zr, err := zip.OpenReader(params.Path)
	if err != nil {
		return nil, archive.ErrUnrecognizedArchiveType
	}

	defer zr.Close()

	var entries []*archive.Entry
	for _, f := range zr.File {
		entries = append(entries, &archive.Entry{
			Name:             archive.CleanFileName(f.Name),
			UncompressedSize: int64(f.UncompressedSize64),
		})
	}

	lr := &ListResult{
		formatName: "Zip", // consistent with 'xad'
		entries:    entries,
	}
	return lr, nil
}

type ListResult struct {
	formatName string
	entries    []*archive.Entry
}

var _ archive.ListResult = (*ListResult)(nil)

func (lr *ListResult) FormatName() string {
	return lr.formatName
}

func (lr *ListResult) Entries() []*archive.Entry {
	return lr.entries
}

func (lr *ListResult) Handler() archive.Handler {
	return &Handler{}
}
