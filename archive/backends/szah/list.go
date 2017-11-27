package szah

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/sevenzip-go/sz"
)

func (h *Handler) List(params *archive.ListParams) (*archive.Contents, error) {
	var entries []*archive.Entry
	err := withArchive(params.Consumer, params.File, func(a *sz.Archive) error {
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

	res := &archive.Contents{
		Entries: entries,
	}
	return res, nil
}
