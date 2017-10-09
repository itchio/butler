package xad

import (
	"encoding/json"
	"os/exec"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
)

func (h *Handler) List(params *archive.ListParams) (archive.ListResult, error) {
	cmd := exec.Command("lsar", "--json", params.Path)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = cmd.Start()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	lsarResult := LsarResult{}
	err = json.NewDecoder(stdout).Decode(&lsarResult)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &ListResult{
		lsarResult: &lsarResult,
	}
	return res, nil
}

type ListResult struct {
	lsarResult *LsarResult
}

var _ archive.ListResult = (*ListResult)(nil)

func (lr *ListResult) FormatName() string {
	return lr.lsarResult.FormatName
}

func (lr *ListResult) Entries() []*archive.Entry {
	var res []*archive.Entry
	for _, entry := range lr.lsarResult.Contents {
		res = append(res, &archive.Entry{
			Name:             archive.CleanFileName(entry.XADFileName),
			UncompressedSize: entry.XADFileSize,
		})
	}
	return res
}

func (lr *ListResult) Handler() archive.Handler {
	return &Handler{}
}
