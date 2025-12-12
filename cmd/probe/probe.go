package probe

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/headway/united"

	"github.com/itchio/savior/countingsource"
	"github.com/itchio/savior/seeksource"

	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/lake/tlc"

	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wire"

	"github.com/pkg/errors"
)

var args = struct {
	patch    string
	fullpath bool
	deep     bool
	dump     string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("probe", "(Advanced) Show statistics about a patch file").Hidden()
	cmd.Arg("patch", "Path of the patch to analyze").Required().StringVar(&args.patch)
	cmd.Flag("fullpath", "Display full path names").BoolVar(&args.fullpath)
	cmd.Flag("deep", "Analyze the top N changed files further").BoolVar(&args.deep)
	cmd.Flag("dump", "Dump ops for any path contain a substring of this").StringVar(&args.dump)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, args.patch))
}

func Do(ctx *mansion.Context, patch string) error {
	patchStats, err := doPrimaryAnalysis(ctx, patch)
	if err != nil {
		return errors.WithStack(err)
	}

	if args.deep {
		err = doDeepAnalysis(ctx, patch, patchStats)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func doPrimaryAnalysis(ctx *mansion.Context, patch string) ([]patchStat, error) {
	consumer := comm.NewStateConsumer()

	patchReader, err := eos.Open(patch, option.WithConsumer(consumer))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	defer patchReader.Close()

	patchSource := seeksource.FromFile(patchReader)

	comm.Opf("patch:  %s", united.FormatBytes(patchSource.Size()))

	cs := countingsource.New(patchSource, func(count int64) {
		comm.Progress(patchSource.Progress())
	})

	_, err = cs.Resume(nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rctx := wire.NewReadContext(cs)
	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	header := &pwr.PatchHeader{}
	err = rctx.ReadMessage(header)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rctx, err = pwr.DecompressWire(rctx, header.Compression)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	target := &tlc.Container{}
	err = rctx.ReadMessage(target)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	source := &tlc.Container{}
	err = rctx.ReadMessage(source)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	comm.Logf("  before: %s in %s", united.FormatBytes(target.Size), target.Stats())
	comm.Logf("   after: %s in %s", united.FormatBytes(target.Size), source.Stats())

	startTime := time.Now()

	comm.StartProgressWithTotalBytes(cs.Size())

	var patchStats []patchStat

	sh := &pwr.SyncHeader{}
	rop := &pwr.SyncOp{}
	bc := &bsdiff.Control{}

	var numBsdiff = 0
	var numRsync = 0
	for fileIndex, f := range source.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		stat := patchStat{
			fileIndex: int64(fileIndex),
			freshData: f.Size,
			algo:      sh.Type,
		}

		if sh.FileIndex != int64(fileIndex) {
			return nil, fmt.Errorf("malformed patch: expected file %d, got %d", fileIndex, sh.FileIndex)
		}

		sourceFile := source.Files[sh.FileIndex]
		doDump := args.dump != "" && strings.Contains(sourceFile.Path, args.dump)

		if doDump {
			consumer.Infof("========== Op Stream Start ===========")
		}

		switch sh.Type {
		case pwr.SyncHeader_RSYNC:
			{
				numRsync++
				readingOps := true
				var pos int64

				for readingOps {
					rop.Reset()

					err = rctx.ReadMessage(rop)
					if err != nil {
						return nil, errors.WithStack(err)
					}

					switch rop.Type {
					case pwr.SyncOp_BLOCK_RANGE:
						tf := target.Files[rop.FileIndex]

						fixedSize := (rop.BlockSpan - 1) * pwr.BlockSize
						lastIndex := rop.BlockIndex + (rop.BlockSpan - 1)
						lastSize := pwr.ComputeBlockSize(tf.Size, lastIndex)
						totalSize := (fixedSize + lastSize)
						stat.freshData -= totalSize
						pos += totalSize
					case pwr.SyncOp_DATA:
						totalSize := int64(len(rop.Data))
						if ctx.Verbose {
							comm.Debugf("%s fresh data at %s (%d-%d)",
								united.FormatBytes(totalSize),
								united.FormatBytes(pos),
								pos, pos+totalSize,
							)
						}
						pos += totalSize
					case pwr.SyncOp_HEY_YOU_DID_IT:
						readingOps = false
					}
				}
			}
		case pwr.SyncHeader_BSDIFF:
			{
				numBsdiff++
				readingOps := true

				bh := &pwr.BsdiffHeader{}
				err = rctx.ReadMessage(bh)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				targetFile := target.Files[bh.TargetIndex]
				if doDump {
					consumer.Infof("It's bsdiff series")
					consumer.Infof("")

					consumer.Infof("Target|index is %d", bh.TargetIndex)
					consumer.Infof("      |path is %s", targetFile.Path)
					consumer.Infof("      |size is %s (%d bytes)", united.FormatBytes(targetFile.Size), targetFile.Size)
					consumer.Infof("")

					consumer.Infof("Source|index is %d", sh.FileIndex)
					consumer.Infof("      |path is %s", sourceFile.Path)
					consumer.Infof("      |size is %s (%d bytes)", united.FormatBytes(sourceFile.Size), sourceFile.Size)
					consumer.Infof("")
				}

				var totalAddBytes int64
				var totalZeroAddBytes int64

				var oldOffset int64
				for readingOps {
					bc.Reset()

					err = rctx.ReadMessage(bc)
					if err != nil {
						return nil, errors.WithStack(err)
					}

					var zeroAddBytes int64
					for _, b := range bc.Add {
						if b == 0 {
							zeroAddBytes++
						}
					}

					totalAddBytes += int64(len(bc.Add))
					totalZeroAddBytes += zeroAddBytes

					stat.freshData -= zeroAddBytes
					if doDump {
						percSimilar := 100.0 * float64(zeroAddBytes) / float64(len(bc.Add))
						if len(bc.Add) == 0 && len(bc.Copy) == 0 {
							// ignore seek
						} else {
							consumer.Infof("Offset: %d\t Add: %d (%.2f%% similar)\tCopy: %d", oldOffset, len(bc.Add), percSimilar, len(bc.Copy))
						}
					}

					oldOffset += int64(len(bc.Add))
					oldOffset += bc.Seek

					if bc.Eof {
						readingOps = false
					}
				}

				if doDump {
					consumer.Statf("Overall: %d/%d add bytes were zero (%.2f%%)", totalZeroAddBytes, totalAddBytes, 100.0*float64(totalZeroAddBytes)/float64(totalAddBytes))
				}

				err = rctx.ReadMessage(rop)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
					msg := fmt.Sprintf("expected HEY_YOU_DID_IT, got %s", rop.Type)
					return nil, errors.New(msg)
				}
			}
		}

		if doDump {
			consumer.Infof("========== Op Stream End ===========")
		}

		patchStats = append(patchStats, stat)
	}

	comm.EndProgress()

	sort.Sort(byDecreasingFreshData(patchStats))

	var totalFresh int64
	for _, stat := range patchStats {
		totalFresh += stat.freshData
	}

	var freshThreshold = int64(0.9 * float64(totalFresh))
	var printedFresh int64

	duration := time.Since(startTime)

	perSec := united.FormatBPS(cs.Size(), duration)
	comm.Statf("Analyzed %s @ %s/s (%s total)", united.FormatBytes(cs.Size()), perSec, duration)
	comm.Statf("%d bsdiff series, %d rsync series", numBsdiff, numRsync)

	var numTouched = 0
	var numTotal = 0
	var naivePatchSize int64
	for _, stat := range patchStats {
		numTotal++
		if stat.freshData > 0 {
			numTouched++
			f := source.Files[stat.fileIndex]
			naivePatchSize += f.Size
		}
	}

	comm.Logf("")
	comm.Statf("Most of the fresh data is in the following files:")

	for i, stat := range patchStats {
		f := source.Files[stat.fileIndex]
		name := f.Path
		if !args.fullpath {
			name = filepath.Base(name)
		}

		comm.Logf("  - %s / %s in %s (%.2f%% changed, %s)",
			united.FormatBytes(stat.freshData),
			united.FormatBytes(f.Size),
			name,
			float64(stat.freshData)/float64(f.Size)*100.0,
			stat.algo)

		printedFresh += stat.freshData

		if i >= 10 || printedFresh >= freshThreshold {
			break
		}
	}

	comm.Logf("")

	var kind = "simple"
	if numBsdiff > 0 {
		kind = "optimized"
	}
	comm.Statf("All in all, that's %s of fresh data in a %s %s patch",
		united.FormatBytes(totalFresh),
		united.FormatBytes(cs.Size()),
		kind,
	)
	comm.Logf(" (%d/%d files are changed by this patch, they weigh a total of %s)", numTouched, numTotal, united.FormatBytes(naivePatchSize))

	return patchStats, nil
}

type deepDiveContext struct {
	target *tlc.Container
	source *tlc.Container
	rctx   *wire.ReadContext

	totalPristine int64
	totalTouched  int64
}

func doDeepAnalysis(ctx *mansion.Context, patch string, patchStats []patchStat) error {
	consumer := comm.NewStateConsumer()

	comm.Logf("")
	var numTouched int
	patchStatPerFileIndex := make(map[int64]patchStat)
	for _, ps := range patchStats {
		patchStatPerFileIndex[ps.fileIndex] = ps
		if ps.freshData > 0 {
			numTouched++
		}
	}

	comm.Statf("Now deep-diving into %d touched files", numTouched)

	patchReader, err := eos.Open(patch, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}

	defer patchReader.Close()

	patchSource := seeksource.FromFile(patchReader)

	cs := countingsource.New(patchSource, func(count int64) {
		comm.Progress(patchSource.Progress())
	})
	_, err = cs.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	comm.Opf("patch:  %s", united.FormatBytes(cs.Size()))

	rctx := wire.NewReadContext(cs)
	err = rctx.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	header := &pwr.PatchHeader{}
	err = rctx.ReadMessage(header)
	if err != nil {
		return errors.WithStack(err)
	}

	rctx, err = pwr.DecompressWire(rctx, header.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	target := &tlc.Container{}
	err = rctx.ReadMessage(target)
	if err != nil {
		return errors.WithStack(err)
	}

	source := &tlc.Container{}
	err = rctx.ReadMessage(source)
	if err != nil {
		return errors.WithStack(err)
	}

	ddc := &deepDiveContext{
		target: target,
		source: source,
		rctx:   rctx,
	}

	sh := &pwr.SyncHeader{}

	for fileIndex := range source.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return errors.WithStack(err)
		}

		if sh.FileIndex != int64(fileIndex) {
			return fmt.Errorf("malformed patch: expected file %d, got %d", fileIndex, sh.FileIndex)
		}

		pc := patchStatPerFileIndex[sh.FileIndex]
		if pc.freshData > 0 {
			err = ddc.analyzeSeries(sh)
		} else {
			err = ddc.skipSeries(sh)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	comm.Statf("All in all, that's %s / %s pristine of all the touched data",
		united.FormatBytes(ddc.totalPristine),
		united.FormatBytes(ddc.totalTouched),
	)

	return nil
}

func (ddc *deepDiveContext) analyzeSeries(sh *pwr.SyncHeader) error {
	f := ddc.source.Files[sh.FileIndex]

	switch sh.Type {
	case pwr.SyncHeader_RSYNC:
		ddc.totalTouched += f.Size
		return ddc.analyzeRsync(sh)
	case pwr.SyncHeader_BSDIFF:
		ddc.totalTouched += f.Size
		return ddc.analyzeBsdiff(sh)
	default:
		return fmt.Errorf("don't know how to analyze series of type %d", sh.Type)
	}
}

func (ddc *deepDiveContext) analyzeRsync(sh *pwr.SyncHeader) error {
	f := ddc.source.Files[sh.FileIndex]
	comm.Debugf("Analyzing rsync series for '%s'", f.Path)

	rctx := ddc.rctx
	readingOps := true

	rop := &pwr.SyncOp{}

	targetBlocks := make(map[int64]int64)

	var pos int64
	var pristine int64

	for readingOps {
		rop.Reset()

		err := rctx.ReadMessage(rop)
		if err != nil {
			return errors.WithStack(err)
		}

		switch rop.Type {
		case pwr.SyncOp_BLOCK_RANGE:
			i := rop.FileIndex
			targetBlocks[i] = targetBlocks[i] + rop.BlockSpan

			tf := ddc.target.Files[rop.FileIndex]

			fixedSize := (rop.BlockSpan - 1) * pwr.BlockSize
			lastIndex := rop.BlockIndex + (rop.BlockSpan - 1)
			lastSize := pwr.ComputeBlockSize(tf.Size, lastIndex)
			totalSize := (fixedSize + lastSize)
			pos += totalSize

			if f.Path == tf.Path {
				if pos == pwr.BlockSize*rop.BlockIndex {
					pristine += totalSize
				}
			}
		case pwr.SyncOp_DATA:
			pos += int64(len(rop.Data))
		case pwr.SyncOp_HEY_YOU_DID_IT:
			readingOps = false
		}
	}

	if len(targetBlocks) > 0 {
		comm.Debugf("Sourcing from '%d' blocks total: ", len(targetBlocks))
		for i, numBlocks := range targetBlocks {
			tf := ddc.target.Files[i]
			comm.Debugf("Taking %d blocks from '%s'", numBlocks, tf.Path)
		}
	} else {
		comm.Debugf("Entirely fresh data!")
	}

	ddc.totalPristine += pristine

	return nil
}

func (ddc *deepDiveContext) analyzeBsdiff(sh *pwr.SyncHeader) error {
	f := ddc.source.Files[sh.FileIndex]
	comm.Debugf("Analyzing bsdiff series for '%s'", f.Path)

	rctx := ddc.rctx
	readingOps := true

	bh := &pwr.BsdiffHeader{}
	err := rctx.ReadMessage(bh)
	if err != nil {
		return errors.WithStack(err)
	}

	tf := ddc.target.Files[bh.TargetIndex]
	comm.Debugf("Diffed against target file '%s'", tf.Path)
	if tf.Path == f.Path {
		comm.Debugf("Same path, can do in-place!")
	}

	bc := &bsdiff.Control{}

	var oldpos int64
	var newpos int64

	var pristine int64
	var similar int64
	var bestUnchanged int64

	clearUnchanged := func() {
		if bestUnchanged > 1024*1024 {
			comm.Debugf("%s contiguous unchanged block ending at from %s to %s",
				united.FormatBytes(bestUnchanged),
				united.FormatBytes(newpos-bestUnchanged),
				united.FormatBytes(newpos),
			)
		}
		bestUnchanged = 0
	}

	var clobbered int64

	for readingOps {
		bc.Reset()

		err = rctx.ReadMessage(bc)
		if err != nil {
			return errors.WithStack(err)
		}

		if bc.Eof {
			readingOps = false
			break
		}

		if len(bc.Add) > 0 {
			if oldpos == newpos {
				var unchanged int64
				for _, b := range bc.Add {
					oldpos++
					newpos++
					if b == 0 {
						unchanged++
						bestUnchanged++
					} else {
						clearUnchanged()
					}
				}
				pristine += unchanged
			} else {
				if oldpos < newpos {
					clobbered += int64(len(bc.Add))
				}
				oldpos += int64(len(bc.Add))
				newpos += int64(len(bc.Add))
			}

			for _, b := range bc.Add {
				if b == 0 {
					similar++
				}
			}
		}

		if len(bc.Copy) > 0 {
			clearUnchanged()
			newpos += int64(len(bc.Copy))
		}

		oldpos += bc.Seek
	}

	rop := &pwr.SyncOp{}

	err = rctx.ReadMessage(rop)
	if err != nil {
		return errors.WithStack(err)
	}

	if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
		msg := fmt.Sprintf("expected HEY_YOU_DID_IT, got %s", rop.Type)
		return errors.New(msg)
	}

	comm.Debugf("%s / %s pristine after patch application", united.FormatBytes(pristine), united.FormatBytes(tf.Size))
	comm.Debugf("File went from %s to %s", united.FormatBytes(tf.Size), united.FormatBytes(f.Size))
	comm.Debugf("%s / %s clobbered total", united.FormatBytes(clobbered), united.FormatBytes(tf.Size))
	comm.Debugf("%s / %s similar total", united.FormatBytes(similar), united.FormatBytes(tf.Size))

	ddc.totalPristine += pristine

	return nil
}

func (ddc *deepDiveContext) skipSeries(sh *pwr.SyncHeader) error {
	rctx := ddc.rctx
	rop := &pwr.SyncOp{}
	bc := &bsdiff.Control{}

	switch sh.Type {
	case pwr.SyncHeader_RSYNC:
		{
			readingOps := true
			for readingOps {
				rop.Reset()

				err := rctx.ReadMessage(rop)
				if err != nil {
					return errors.WithStack(err)
				}

				if rop.Type == pwr.SyncOp_HEY_YOU_DID_IT {
					// yay, we did it!
					readingOps = false
				}
			}
		}
	case pwr.SyncHeader_BSDIFF:
		{
			bh := &pwr.BsdiffHeader{}
			err := rctx.ReadMessage(bh)
			if err != nil {
				return errors.WithStack(err)
			}

			readingOps := true
			for readingOps {
				bc.Reset()

				err := rctx.ReadMessage(bc)
				if err != nil {
					return errors.WithStack(err)
				}

				if bc.Eof {
					readingOps = false
				}
			}

			rop.Reset()
			err = rctx.ReadMessage(rop)
			if err != nil {
				return errors.WithStack(err)
			}

			if rop.Type != pwr.SyncOp_HEY_YOU_DID_IT {
				// oh noes, we didn't do it
				return errors.New("missing HEY_YOU_DID_IT after bsdiff series")
			}
		}
	default:
		return fmt.Errorf("dunno how to skip series of type %d", sh.Type)
	}

	return nil
}

type patchStat struct {
	fileIndex int64
	freshData int64
	algo      pwr.SyncHeader_Type
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
