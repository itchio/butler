package naked

import (
	"io"
	"os"
	"path/filepath"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	stats, err := params.File.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	destName := filepath.Base(stats.Name())
	destAbsolutePath := filepath.Join(params.InstallFolderPath, destName)

	params.Consumer.Infof("Creating %s", params.InstallFolderPath)
	err = os.MkdirAll(params.InstallFolderPath, 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	params.Consumer.Infof("Writing %s", destAbsolutePath)

	w, err := os.Create(destAbsolutePath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	// TODO: something smarter - resume for starters, multi-segment download maybe,
	// idk but just `io.Copy` feels wrong

	startTime := time.Now()
	writtenBytes, err := io.Copy(w, params.File)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	params.Consumer.Infof("Fetched %s in %s", humanize.IBytes(uint64(writtenBytes)), time.Since(startTime))

	var res = installer.InstallResult{
		Files: []string{
			destName,
		},
	}

	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,
		Consumer: params.Consumer,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &res, nil
}
