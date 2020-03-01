package operate

import (
	"os"
	"path/filepath"

	"github.com/itchio/hush"
	"github.com/itchio/hush/bfs"
	"github.com/itchio/httpkit/eos"
	"github.com/pkg/errors"
)

type Accuracy int

const (
	AccuracyNone     Accuracy = 0
	AccuracyGuess    Accuracy = 10
	AccuracyComputed Accuracy = 20
)

func (a Accuracy) String() string {
	switch a {
	case AccuracyNone:
		return "no information"
	case AccuracyGuess:
		return "guess"
	case AccuracyComputed:
		return "computed"
	default:
		return "<?>"
	}
}

type DiskUsageInfo struct {
	// Space we'll use once the install is all said and done
	FinalDiskUsage int64

	// Space we need to perform that operation
	NeededFreeSpace int64

	// Accuracy of our information
	Accuracy Accuracy
}

func AssessDiskUsage(sourceFile eos.File, receiptIn *bfs.Receipt, installFolder string, installerInfo *hush.InstallerInfo) (*DiskUsageInfo, error) {
	dui := &DiskUsageInfo{
		Accuracy: AccuracyNone,
	}

	if installerInfo.Type == hush.InstallerTypeNaked {
		// for naked installers, we can tell exactly how much space we'll need!
		dui.Accuracy = AccuracyComputed

		stats, err := sourceFile.Stat()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		dui.NeededFreeSpace = stats.Size()
		dui.FinalDiskUsage = stats.Size()
		return dui, nil
	}

	existingFiles := make(map[string]struct{})
	if receiptIn.HasFiles() {
		for _, rf := range receiptIn.Files {
			existingFiles[filepath.ToSlash(filepath.Clean(rf))] = struct{}{}
		}
	}

	if len(installerInfo.Entries) > 0 {
		dui.Accuracy = AccuracyComputed
		for _, e := range installerInfo.Entries {
			dui.FinalDiskUsage += e.UncompressedSize

			var existingSize int64
			if _, ok := existingFiles[e.CanonicalPath]; ok {
				rfs, err := os.Stat(filepath.Join(installFolder, e.CanonicalPath))
				if err == nil {
					// oh well
					existingSize = rfs.Size()
				}
			}
			dui.NeededFreeSpace += e.UncompressedSize - existingSize
		}
		return dui, nil
	}

	// if we have no entries listed, let's take a guess
	dui.Accuracy = AccuracyGuess
	stats, err := sourceFile.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// let's assume the uncompressed game is 1.3x as
	// large as the install source. this could be completely
	// inaccurate in either direction.
	downloadSize := stats.Size()
	installSize := downloadSize * 130 / 100

	dui.NeededFreeSpace = downloadSize + installSize
	dui.FinalDiskUsage = installSize

	return dui, nil
}
