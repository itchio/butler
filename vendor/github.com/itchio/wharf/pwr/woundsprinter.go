package pwr

import (
	"fmt"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/tlc"
)

type WoundsPrinter struct {
	Wounds chan *Wound
}

func (wp *WoundsPrinter) Do(signature *SignatureInfo, consumer *StateConsumer) error {
	if consumer == nil {
		return fmt.Errorf("Missing Consumer in WoundsPrinter")
	}

	for wound := range wp.Wounds {
		consumer.Infof(wound.PrettyString(signature.Container))
	}

	return nil
}

func (w *Wound) PrettyString(container *tlc.Container) string {
	file := container.Files[w.FileIndex]
	woundSize := humanize.IBytes(uint64(w.End - w.Start))
	offset := humanize.IBytes(uint64(w.Start))
	return fmt.Sprintf("~%s wound %s into %s", woundSize, offset, file.Path)
}
