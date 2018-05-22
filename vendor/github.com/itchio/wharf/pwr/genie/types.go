package genie

import (
	"fmt"

	"github.com/itchio/httpkit/progress"
)

type BlockOrigin struct {
	FileIndex int64
	Offset    int64
	Size      int64
}

func (bo *BlockOrigin) GetSize() int64 {
	return bo.Size
}

type FreshOrigin struct {
	Size int64
}

func (fo *FreshOrigin) GetSize() int64 {
	return fo.Size
}

type Origin interface {
	GetSize() int64
}

type Composition struct {
	FileIndex  int64
	BlockIndex int64
	Origins    []Origin
	Size       int64
}

func (comp *Composition) Append(origin Origin) {
	comp.Origins = append(comp.Origins, origin)
	comp.Size += origin.GetSize()
}

func (comp *Composition) String() string {
	res := fmt.Sprintf("file %d, block %d (%s) is composed of: ", comp.FileIndex, comp.BlockIndex, progress.FormatBytes(comp.Size))
	for i, origin := range comp.Origins {
		if i > 0 {
			res += ", "
		}
		res += fmt.Sprintf("%+v", origin)
	}
	return res
}
