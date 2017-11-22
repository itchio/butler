package szah

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/sevenzip-go/sz"
)

type ech struct {
	params *archive.ExtractParams
}

func (h *Handler) Extract(params *archive.ExtractParams) error {
	err := withArchive(params.Path, func(a *sz.Archive) error {
		itemCount, err := a.GetItemCount()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		var totalUncompressedSize int64

		indices := make([]int64, itemCount)
		for i := int64(0); i < itemCount; i++ {
			indices[i] = i

			func() {
				item := a.GetItem(i)
				if item == nil {
					return
				}
				defer item.Free()

				totalUncompressedSize += int64(item.GetUInt64Property(sz.PidSize))
			}()
		}

		if params.OnUncompressedSizeKnown != nil {
			params.OnUncompressedSizeKnown(totalUncompressedSize)
		}

		ec, err := sz.NewExtractCallback(&ech{
			params: params,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer ec.Free()

		err = a.ExtractSeveral(indices, ec)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (e *ech) GetStream(item *sz.Item) (*sz.OutStream, error) {
	sanePath := sanitizePath(item.GetStringProperty(sz.PidPath))
	outPath := filepath.Join(e.params.OutputPath, sanePath)

	if item.GetBoolProperty(sz.PidIsDir) {
		err := os.MkdirAll(outPath, 0755)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return nil, nil
	}

	err := os.MkdirAll(filepath.Dir(outPath), 0755)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return sz.NewOutStream(f)
}

func (e *ech) SetProgress(complete int64, total int64) {
	if total > 0 {
		e.params.Consumer.Progress(float64(complete) / float64(total))
	}
	// TODO: do something smart for other formats ?
}

func sanitizePath(inPath string) string {
	outPath := filepath.ToSlash(inPath)

	if runtime.GOOS == "windows" {
		// Remove illegal character for windows paths, see
		// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
		for i := byte(0); i <= 31; i++ {
			outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
		}
	}

	return outPath
}
