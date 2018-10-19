package pwr

import (
	"fmt"
	"io"
	"time"

	"path/filepath"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

// FileOrigin maps a target's file index to how many bytes it
// contribute to a given source file
type FileOrigin map[int64]int64

// A DiffMapping is a pair of files that have similar contents (blocks in common)
// or equal paths, and which are good candidates for bsdiffing
type DiffMapping struct {
	TargetIndex int64
	NumBytes    int64
}

// DiffMappings contains one diff mapping for each pair of files to be bsdiff'd
type DiffMappings map[int64]*DiffMapping

// ToString returns a human-readable representation of all diff mappings,
// which gives an overview of how files changed.
func (dm DiffMappings) ToString(sourceContainer tlc.Container, targetContainer tlc.Container) string {
	s := ""
	for sourceIndex, diffMapping := range dm {
		s += fmt.Sprintf("%s <- %s (%s in common)\n",
			sourceContainer.Files[sourceIndex].Path,
			targetContainer.Files[diffMapping.TargetIndex].Path,
			progress.FormatBytes(diffMapping.NumBytes),
		)
	}
	return s
}

// A Timeline contains time-coded events pertaining to the rediff process
type Timeline struct {
	Groups []TimelineGroup `json:"groups"`
	Items  []TimelineItem  `json:"items"`
}

// A TimelineGroup is what timeline items are grouped by. All items
// of a given group appear in the same row.
type TimelineGroup struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

// A TimelineItem represents a task that occured in a certain period of time
type TimelineItem struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Content string  `json:"content"`
	Style   string  `json:"style"`
	Title   string  `json:"title"`
	Group   int     `json:"group"`
}

// RediffContext holds options for the rediff process, along with
// some state.
type RediffContext struct {
	SourcePool wsync.Pool
	TargetPool wsync.Pool

	////////////////////
	// optional
	////////////////////

	// RediffSizeLimit is the maximum size of a file we'll attempt to rediff.
	// If a file is larger than that, ops will just be copied.
	RediffSizeLimit       int64
	SuffixSortConcurrency int
	Partitions            int
	Compression           *CompressionSettings
	Consumer              *state.Consumer
	BsdiffStats           *bsdiff.DiffStats
	Timeline              *Timeline
	ForceMapAll           bool

	////////////////////
	// set on Analyze
	////////////////////

	TargetContainer *tlc.Container
	SourceContainer *tlc.Container

	////////////////////
	// internal
	////////////////////

	DiffMappings DiffMappings
	MeasureMem   bool
}

const DefaultRediffSizeLimit = 4 * 1024 * 1024 * 1024 // 4GB

// AnalyzePatch parses a non-optimized patch, looking for good bsdiff'ing candidates
// and building DiffMappings.
func (rc *RediffContext) AnalyzePatch(patchReader savior.SeekSource) error {
	var err error

	if rc.RediffSizeLimit == 0 {
		rc.RediffSizeLimit = DefaultRediffSizeLimit
	}

	rctx := wire.NewReadContext(patchReader)

	err = rctx.ExpectMagic(PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	ph := &PatchHeader{}
	err = rctx.ReadMessage(ph)
	if err != nil {
		return errors.WithStack(err)
	}

	rctx, err = DecompressWire(rctx, ph.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	targetContainer := &tlc.Container{}
	err = rctx.ReadMessage(targetContainer)
	if err != nil {
		return errors.WithStack(err)
	}
	rc.TargetContainer = targetContainer

	sourceContainer := &tlc.Container{}
	err = rctx.ReadMessage(sourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}
	rc.SourceContainer = sourceContainer

	rop := &SyncOp{}

	targetPathsToIndex := make(map[string]int64)
	for targetFileIndex, file := range targetContainer.Files {
		targetPathsToIndex[file.Path] = int64(targetFileIndex)
	}

	rc.DiffMappings = make(DiffMappings)

	var doneBytes int64

	sh := &SyncHeader{}

	for sourceFileIndex, sourceFile := range sourceContainer.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return errors.WithStack(err)
		}

		if sh.FileIndex != int64(sourceFileIndex) {
			return errors.WithStack(fmt.Errorf("Malformed patch, expected index %d, got %d", sourceFileIndex, sh.FileIndex))
		}

		rc.Consumer.ProgressLabel(sourceFile.Path)
		rc.Consumer.Progress(float64(doneBytes) / float64(sourceContainer.Size))

		bytesReusedPerFileIndex := make(FileOrigin)
		readingOps := true
		var numBlockRange int64
		var numData int64

		for readingOps {
			rop.Reset()
			err = rctx.ReadMessage(rop)
			if err != nil {
				return errors.WithStack(err)
			}

			switch rop.Type {
			case SyncOp_BLOCK_RANGE:
				numBlockRange++
				alreadyReused := bytesReusedPerFileIndex[rop.FileIndex]
				lastBlockIndex := rop.BlockIndex + rop.BlockSpan
				targetFile := targetContainer.Files[rop.FileIndex]
				lastBlockSize := ComputeBlockSize(targetFile.Size, lastBlockIndex)
				otherBlocksSize := BlockSize*rop.BlockSpan - 1

				bytesReusedPerFileIndex[rop.FileIndex] = alreadyReused + otherBlocksSize + lastBlockSize

			case SyncOp_DATA:
				numData++

			default:
				switch rop.Type {
				case SyncOp_HEY_YOU_DID_IT:
					readingOps = false
				default:
					return errors.WithStack(ErrMalformedPatch)
				}
			}
		}

		if numBlockRange == 1 && numData == 0 && !rc.ForceMapAll {
			// transpositions (renames, etc.) don't need bsdiff'ing :)
		} else {
			var diffMapping *DiffMapping

			for targetFileIndex, numBytes := range bytesReusedPerFileIndex {
				targetFile := targetContainer.Files[targetFileIndex]
				// first, better, or equal target file with same name (prefer natural mappings)
				if diffMapping == nil || numBytes > diffMapping.NumBytes || (numBytes == diffMapping.NumBytes && targetFile.Path == sourceFile.Path) {
					diffMapping = &DiffMapping{
						TargetIndex: targetFileIndex,
						NumBytes:    numBytes,
					}
				}
			}

			if diffMapping == nil {
				// even without any common blocks, bsdiff might still be worth it
				// if the file is named the same
				if samePathTargetFileIndex, ok := targetPathsToIndex[sourceFile.Path]; ok {
					targetFile := targetContainer.Files[samePathTargetFileIndex]

					// don't take into account files that were 0 bytes (it happens). bsdiff won't like that.
					if targetFile.Size > 0 {
						diffMapping = &DiffMapping{
							TargetIndex: samePathTargetFileIndex,
							NumBytes:    0,
						}
					}
				}
			}

			if sourceFile.Size > rc.RediffSizeLimit {
				// source file is too large, skip rediff
				diffMapping = nil
			}

			if diffMapping != nil {
				targetFile := targetContainer.Files[diffMapping.TargetIndex]
				if targetFile.Size > rc.RediffSizeLimit {
					// target file is too large, skip rediff
					diffMapping = nil
				}
			}

			if diffMapping != nil {
				rc.DiffMappings[int64(sourceFileIndex)] = diffMapping
			}
		}

		doneBytes += sourceFile.Size
	}

	return nil
}

// OptimizePatch uses the information computed by AnalyzePatch to write a new version of
// the patch, but with bsdiff instead of rsync diffs for each DiffMapping.
func (rc *RediffContext) OptimizePatch(patchReader savior.SeekSource, patchWriter io.Writer) error {
	var err error

	if rc.SourcePool == nil {
		return errors.WithStack(fmt.Errorf("SourcePool cannot be nil"))
	}

	if rc.TargetPool == nil {
		return errors.WithStack(fmt.Errorf("TargetPool cannot be nil"))
	}

	if rc.DiffMappings == nil {
		return errors.WithStack(fmt.Errorf("AnalyzePatch must be called before OptimizePatch"))
	}

	rctx := wire.NewReadContext(patchReader)
	wctx := wire.NewWriteContext(patchWriter)

	err = wctx.WriteMagic(PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	err = rctx.ExpectMagic(PatchMagic)
	if err != nil {
		return errors.WithStack(err)
	}

	ph := &PatchHeader{}
	err = rctx.ReadMessage(ph)
	if err != nil {
		return errors.WithStack(err)
	}

	compression := rc.Compression
	if compression == nil {
		compression = defaultRediffCompressionSettings()
	}

	wph := &PatchHeader{
		Compression: compression,
	}
	err = wctx.WriteMessage(wph)
	if err != nil {
		return errors.WithStack(err)
	}

	rctx, err = DecompressWire(rctx, ph.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	wctx, err = CompressWire(wctx, wph.Compression)
	if err != nil {
		return errors.WithStack(err)
	}

	targetContainer := &tlc.Container{}
	err = rctx.ReadMessage(targetContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	err = wctx.WriteMessage(targetContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	sourceContainer := &tlc.Container{}
	err = rctx.ReadMessage(sourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	err = wctx.WriteMessage(sourceContainer)
	if err != nil {
		return errors.WithStack(err)
	}

	sh := &SyncHeader{}
	bh := &BsdiffHeader{}
	rop := &SyncOp{}

	bdc := &bsdiff.DiffContext{
		SuffixSortConcurrency: rc.SuffixSortConcurrency,
		Partitions:            rc.Partitions,
		Stats:                 rc.BsdiffStats,
		MeasureMem:            rc.MeasureMem,
	}

	if rc.Timeline != nil {
		rc.Timeline.Groups = append(rc.Timeline.Groups, TimelineGroup{
			ID:      0,
			Content: "Worker",
		})
	}

	initialStart := time.Now()
	bconsumer := &state.Consumer{}

	var biggestSourceFile int64
	var totalRediffSize int64

	for sourceFileIndex, sourceFile := range sourceContainer.Files {
		if _, ok := rc.DiffMappings[int64(sourceFileIndex)]; ok {
			if sourceFile.Size > biggestSourceFile {
				biggestSourceFile = sourceFile.Size
			}

			totalRediffSize += sourceFile.Size
		}
	}

	var doneSize int64

	for sourceFileIndex, sourceFile := range sourceContainer.Files {
		sh.Reset()
		err = rctx.ReadMessage(sh)
		if err != nil {
			return errors.WithStack(err)
		}

		if sh.FileIndex != int64(sourceFileIndex) {
			return errors.WithStack(fmt.Errorf("Malformed patch, expected index %d, got %d", sourceFileIndex, sh.FileIndex))
		}

		diffMapping := rc.DiffMappings[int64(sourceFileIndex)]

		if diffMapping == nil {
			// if no mapping, just copy ops straight up
			err = wctx.WriteMessage(sh)
			if err != nil {
				return errors.WithStack(err)
			}

			for {
				rop.Reset()
				err = rctx.ReadMessage(rop)
				if err != nil {
					return errors.WithStack(err)
				}

				if rop.Type == SyncOp_HEY_YOU_DID_IT {
					break
				}

				err = wctx.WriteMessage(rop)
				if err != nil {
					return errors.WithStack(err)
				}
			}
		} else {
			// signal bsdiff start to patcher
			sh.Reset()
			sh.FileIndex = int64(sourceFileIndex)
			sh.Type = SyncHeader_BSDIFF
			err = wctx.WriteMessage(sh)
			if err != nil {
				return errors.WithStack(err)
			}

			bh.Reset()
			bh.TargetIndex = diffMapping.TargetIndex
			err = wctx.WriteMessage(bh)
			if err != nil {
				return errors.WithStack(err)
			}

			// throw away old ops
			for {
				err = rctx.ReadMessage(rop)
				if err != nil {
					return errors.WithStack(err)
				}

				if rop.Type == SyncOp_HEY_YOU_DID_IT {
					break
				}
			}

			// then bsdiff
			sourceFileReader, err := rc.SourcePool.GetReadSeeker(int64(sourceFileIndex))
			if err != nil {
				return errors.WithStack(err)
			}

			targetFileReader, err := rc.TargetPool.GetReadSeeker(diffMapping.TargetIndex)
			if err != nil {
				return errors.WithStack(err)
			}

			rc.Consumer.ProgressLabel(fmt.Sprintf(">%s", sourceFile.Path))

			_, err = sourceFileReader.Seek(0, io.SeekStart)
			if err != nil {
				return errors.WithStack(err)
			}

			rc.Consumer.ProgressLabel(fmt.Sprintf("<%s", sourceFile.Path))

			_, err = targetFileReader.Seek(0, io.SeekStart)
			if err != nil {
				return errors.WithStack(err)
			}

			rc.Consumer.ProgressLabel(fmt.Sprintf("*%s", sourceFile.Path))

			startTime := time.Now()

			err = bdc.Do(targetFileReader, sourceFileReader, wctx.WriteMessage, bconsumer)

			endTime := time.Now()

			if rc.Timeline != nil {
				heat := int(float64(sourceFile.Size) / float64(biggestSourceFile) * 240.0)
				rc.Timeline.Items = append(rc.Timeline.Items, TimelineItem{
					Content: filepath.Base(sourceFile.Path),
					Style:   fmt.Sprintf("background-color: hsl(%d, 100%%, 50%%)", heat),
					Title:   fmt.Sprintf("%s %s", progress.FormatBytes(sourceFile.Size), sourceFile.Path),
					Start:   startTime.Sub(initialStart).Seconds(),
					End:     endTime.Sub(initialStart).Seconds(),
					Group:   0,
				})
			}

			doneSize += sourceFile.Size
		}

		// and don't forget to indicate success
		rop.Reset()
		rop.Type = SyncOp_HEY_YOU_DID_IT

		err = wctx.WriteMessage(rop)
		if err != nil {
			return errors.WithStack(err)
		}

		rc.Consumer.Progress(float64(doneSize) / float64(totalRediffSize))
	}

	err = wctx.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func defaultRediffCompressionSettings() *CompressionSettings {
	return &CompressionSettings{
		Algorithm: CompressionAlgorithm_BROTLI,
		Quality:   9,
	}
}
