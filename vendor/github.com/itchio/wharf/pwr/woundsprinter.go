package pwr

import (
	"fmt"

	humanize "github.com/dustin/go-humanize"
)

type WoundsPrinter struct {
	Wounds chan *Wound
}

func (wp *WoundsPrinter) Do(signature *SignatureInfo, consumer *StateConsumer) error {
	if consumer == nil {
		return fmt.Errorf("Missing Consumer in WoundsPrinter")
	}

	for wound := range wp.Wounds {
		file := signature.Container.Files[wound.FileIndex]
		woundSize := humanize.IBytes(uint64(wound.End - wound.Start))
		offset := humanize.IBytes(uint64(wound.Start))
		consumer.Infof("~%s wound %s into %s", woundSize, offset, file.Path)
	}

	return nil
}
