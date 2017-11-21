package szah

import (
	"github.com/fasterthanlime/go-libc7zip/sz"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
)

func (h *Handler) List(params *archive.ListParams) (archive.ListResult, error) {
	var entries []*archive.Entry
	err := withArchive(params.Path, func(a *sz.Archive) error {
		itemCount, err := a.GetItemCount()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for i := int64(0); i < itemCount; i++ {
			func() {
				item := a.GetItem(i)
				if item == nil {
					return
				}
				defer item.Free()

				sanePath := sanitizePath(item.GetStringProperty(sz.PidPath))

				entries = append(entries, &archive.Entry{
					Name:             sanePath,
					UncompressedSize: int64(item.GetUInt64Property(sz.PidSize)),
				})
			}()
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	lr := &ListResult{
		formatName: "generic", // TODO: maybe remove that if it's unused
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
