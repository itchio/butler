package main

import (
	"fmt"
	"sort"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

func probe(patch string) {
	must(doProbe(patch))
}

func doProbe(patch string) error {
	patchReader, err := eos.Open(patch)
	if err != nil {
		return err
	}

	defer patchReader.Close()

	stats, err := patchReader.Stat()
	if err != nil {
		return err
	}

	comm.Statf("patch:  %s", humanize.IBytes(uint64(stats.Size())))

	rctx := wire.NewReadContext(patchReader)
	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return err
	}

	header := &pwr.PatchHeader{}
	err = rctx.ReadMessage(header)
	if err != nil {
		return err
	}

	rctx, err = pwr.DecompressWire(rctx, header.Compression)
	if err != nil {
		return err
	}

	target := &tlc.Container{}
	err = rctx.ReadMessage(target)
	if err != nil {
		return err
	}

	source := &tlc.Container{}
	err = rctx.ReadMessage(source)
	if err != nil {
		return err
	}

	comm.Statf("target: %s in %s", humanize.IBytes(uint64(target.Size)), target.Stats())
	comm.Statf("source: %s in %s", humanize.IBytes(uint64(target.Size)), source.Stats())

	var patchStats []patchStat

	sh := &pwr.SyncHeader{}
	rop := &pwr.SyncOp{}

	for fileIndex, f := range source.Files {
		stat := patchStat{
			fileIndex: int64(fileIndex),
			freshData: f.Size,
		}

		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return err
		}

		if sh.FileIndex != int64(fileIndex) {
			return fmt.Errorf("malformed patch: expected file %d, got %d", fileIndex, sh.FileIndex)
		}

		readingOps := true

		var pos int64

		for readingOps {
			rop.Reset()

			err = rctx.ReadMessage(rop)
			if err != nil {
				return err
			}

			switch rop.Type {
			case pwr.SyncOp_BLOCK_RANGE:
				fixedSize := (rop.BlockSpan - 1) * pwr.BlockSize
				lastIndex := rop.BlockIndex + (rop.BlockSpan - 1)
				lastSize := pwr.ComputeBlockSize(f.Size, lastIndex)
				totalSize := (fixedSize + lastSize)
				stat.freshData -= totalSize
				pos += totalSize
			case pwr.SyncOp_DATA:
				totalSize := int64(len(rop.Data))
				if *appArgs.verbose {
					comm.Debugf("%s fresh data at %s (%d-%d)", humanize.IBytes(uint64(totalSize)), humanize.IBytes(uint64(pos)),
						pos, pos+totalSize)
				}
				pos += totalSize
			case pwr.SyncOp_HEY_YOU_DID_IT:
				readingOps = false
			}
		}

		patchStats = append(patchStats, stat)
	}

	sort.Sort(byDecreasingFreshData(patchStats))

	var totalFresh int64
	for _, stat := range patchStats {
		totalFresh += stat.freshData
	}

	var eightyFresh = int64(0.8 * float64(totalFresh))
	var printedFresh int64

	comm.Opf("80%% of fresh data is in the following files:")

	for _, stat := range patchStats {
		f := source.Files[stat.fileIndex]
		comm.Logf("%s in %s (%.2f%% changed)",
			humanize.IBytes(uint64(stat.freshData)),
			f.Path,
			float64(stat.freshData)/float64(f.Size)*100.0)

		printedFresh += stat.freshData
		if printedFresh >= eightyFresh {
			break
		}
	}

	return nil
}

type patchStat struct {
	fileIndex int64
	freshData int64
}

type byDecreasingFreshData []patchStat

func (s byDecreasingFreshData) Len() int {
	return len(s)
}

func (s byDecreasingFreshData) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byDecreasingFreshData) Less(i, j int) bool {
	return s[j].freshData < s[i].freshData
}
