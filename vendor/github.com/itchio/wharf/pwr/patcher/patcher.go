package patcher

import (
	"fmt"
	"io"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/wharf/bsdiff"

	"github.com/itchio/savior"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wire"
	"github.com/itchio/wharf/wsync"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

type savingPatcher struct {
	rctx     *wire.ReadContext
	consumer *state.Consumer

	sc SaveConsumer

	targetContainer *tlc.Container
	sourceContainer *tlc.Container
	header          *pwr.PatchHeader

	rsyncCtx  *wsync.Context
	bsdiffCtx *bsdiff.PatchContext
}

var _ Patcher = (*savingPatcher)(nil)

// New reads the patch header and returns a patcher that
// is ready to Resume, either from the start (nil checkpoint)
// or partway through the patch
func New(patchReader savior.SeekSource, consumer *state.Consumer) (Patcher, error) {
	// Reading the header & both containers is done even
	// when we resume patching partway through (from a checkpoint)
	// Downside: more network usage when resuming
	// Upside: no need to store that on disk

	startOffset, err := patchReader.Resume(nil)
	if err != nil {
		return nil, err
	}

	if startOffset != 0 {
		return nil, errors.Errorf("expected source to resume at 0, got %d", startOffset)
	}

	rawWire := wire.NewReadContext(patchReader)

	// Ensure magic

	err = rawWire.ExpectMagic(pwr.PatchMagic)
	if err != nil {
		return nil, err
	}

	// Read header & decompress if needed

	header := &pwr.PatchHeader{}
	err = rawWire.ReadMessage(header)
	if err != nil {
		return nil, err
	}

	rctx, err := pwr.DecompressWire(rawWire, header.Compression)
	if err != nil {
		return nil, err
	}

	// Read both containers

	targetContainer := &tlc.Container{}
	err = rctx.ReadMessage(targetContainer)
	if err != nil {
		return nil, err
	}

	sourceContainer := &tlc.Container{}
	err = rctx.ReadMessage(sourceContainer)
	if err != nil {
		return nil, err
	}

	consumer.Debugf("→ Created patcher")
	consumer.Debugf("before: %s", targetContainer.Stats())
	consumer.Debugf(" after: %s", sourceContainer.Stats())

	sp := &savingPatcher{
		rctx:     rctx,
		consumer: consumer,

		targetContainer: targetContainer,
		sourceContainer: sourceContainer,
		header:          header,
	}

	return sp, nil
}

func (sp *savingPatcher) Resume(c *Checkpoint, targetPool wsync.Pool, bowl bowl.Bowl) error {
	if sp.sc == nil {
		sp.sc = &nopSaveConsumer{}
	}

	consumer := sp.consumer

	if c != nil {
		err := sp.rctx.Resume(c.MessageCheckpoint)
		if err != nil {
			return err
		}
	} else {
		c = &Checkpoint{
			FileIndex: 0,
		}
	}

	var numFiles = int64(len(sp.sourceContainer.Files))
	consumer.Debugf("↺ Resuming from file %d / %d", c.FileIndex, numFiles)

	for c.FileIndex < numFiles {
		f := sp.sourceContainer.Files[c.FileIndex]
		var sh *pwr.SyncHeader

		consumer.Debugf("→ Patching #%d: '%s'", c.FileIndex, f.Path)

		if c.SyncHeader != nil {
			sh = c.SyncHeader
			consumer.Debugf("...from checkpoint")
		} else {
			sh = &pwr.SyncHeader{}

			err := sp.rctx.ReadMessage(sh)
			if err != nil {
				return err
			}

			if sh.FileIndex != c.FileIndex {
				return errors.Errorf("corrupted patch or internal error: expected file %d, got file %d", c.FileIndex, sh.FileIndex)
			}

			switch sh.Type {
			case pwr.SyncHeader_RSYNC:
				c.FileKind = FileKindRsync
			case pwr.SyncHeader_BSDIFF:
				c.FileKind = FileKindBsdiff
			default:
				return errors.Errorf("unknown patch series kind %d for '%s'", sh.Type, f.Path)
			}
		}

		err := sp.processFile(c, targetPool, sh, bowl)
		if err != nil {
			return err
		}

		// reset checkpoint and increment
		c.FileIndex++
		c.RsyncCheckpoint = nil
		c.BsdiffCheckpoint = nil
		c.MessageCheckpoint = nil
		c.SyncHeader = nil
	}

	return nil
}

func (sp *savingPatcher) processFile(c *Checkpoint, targetPool wsync.Pool, sh *pwr.SyncHeader, bwl bowl.Bowl) error {
	switch c.FileKind {
	case FileKindRsync:
		return sp.processRsync(c, targetPool, sh, bwl)
	case FileKindBsdiff:
		return sp.processBsdiff(c, targetPool, sh, bwl)
	default:
		return errors.Errorf("unknown file kind %d", sh.Type)
	}
}

func (sp *savingPatcher) processRsync(c *Checkpoint, targetPool wsync.Pool, sh *pwr.SyncHeader, bwl bowl.Bowl) error {
	var op *pwr.SyncOp

	var writer bowl.EntryWriter

	if c.RsyncCheckpoint != nil {
		var err error

		// alrighty let's do it
		writer, err = bwl.GetWriter(sh.FileIndex)
		if err != nil {
			return err
		}

		// FIXME: swallowed error
		defer writer.Close()

		_, err = writer.Resume(c.RsyncCheckpoint.BowlCheckpoint)
		if err != nil {
			return err
		}

		f := sp.sourceContainer.Files[sh.FileIndex]
		sp.consumer.Debugf("↺ Resuming rsync entry @ %s / %s",
			humanize.IBytes(uint64(writer.Tell())),
			humanize.IBytes(uint64(f.Size)),
		)
	} else {
		// starting from the beginning!

		// let's see if it's a transposition
		op = &pwr.SyncOp{}
		err := sp.rctx.ReadMessage(op)
		if err != nil {
			return err
		}

		if sp.isFullFileOp(sh, op) {
			// oh dang it's either a true no-op, or a rename.
			// either way, we're not troubling the rsync patcher
			// for that.
			sp.consumer.Debugf("Transpose: '%s' -> '%s'",
				sp.targetContainer.Files[op.FileIndex].Path,
				sp.sourceContainer.Files[op.FileIndex].Path,
			)

			err := bwl.Transpose(bowl.Transposition{
				SourceIndex: sh.FileIndex,
				TargetIndex: op.FileIndex,
			})
			if err != nil {
				return err
			}

			// however, we do have to read the end marker
		readUntilEndMarker:
			for {
				err = sp.rctx.ReadMessage(op)
				if err != nil {
					return err
				}

				if op.Type == pwr.SyncOp_HEY_YOU_DID_IT {
					break readUntilEndMarker
				}

				// butler used to emit a 0-len DATA op after
				// full-file ops in some conditions. we can
				// ignore them.
				sp.consumer.Debugf("%s has trailing %s op after full-file op", sp.sourceContainer.Files[sh.FileIndex].Path, op.Type)
			}

			return nil
		}

		// not a full-file op, let's patch!
		writer, err = bwl.GetWriter(sh.FileIndex)
		if err != nil {
			return err
		}

		// FIXME: swallowed error
		defer writer.Close()

		_, err = writer.Resume(nil)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if sp.rsyncCtx == nil {
		sp.rsyncCtx = wsync.NewContext(int(pwr.BlockSize))
	}

	if op == nil {
		// we resumed somewhere in the middle, let's initialize op
		op = &pwr.SyncOp{}
	} else {
		// op is non-nil, so we started from scratch and
		// the first op was not a full-file op.
		// We want to relay it now

		wop, err := makeWop(op)
		if err != nil {
			return errors.WithStack(err)
		}

		err = sp.rsyncCtx.ApplySingle(
			writer,
			targetPool,
			wop,
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// let's relay the rest of the messages!
	for {
		if sp.sc.ShouldSave() {
			sp.rctx.WantSave()

			messageCheckpoint := sp.rctx.PopCheckpoint()
			if messageCheckpoint != nil {
				bowlCheckpoint, err := writer.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				// oh damn it's our time
				checkpoint := &Checkpoint{
					SyncHeader:        sh,
					FileIndex:         sh.FileIndex,
					FileKind:          FileKindRsync,
					MessageCheckpoint: messageCheckpoint,
					RsyncCheckpoint: &RsyncCheckpoint{
						BowlCheckpoint: bowlCheckpoint,
					},
				}
				action, err := sp.sc.Save(checkpoint)
				if err != nil {
					return errors.WithStack(err)
				}

				switch action {
				case AfterSaveStop:
					return errors.WithStack(ErrStop)
				}
			}
		}

		err := sp.rctx.ReadMessage(op)
		if err != nil {
			return errors.WithStack(err)
		}

		if op.Type == pwr.SyncOp_HEY_YOU_DID_IT {
			// hey, we did it!
			return nil
		}

		wop, err := makeWop(op)
		if err != nil {
			return errors.WithStack(err)
		}

		err = sp.rsyncCtx.ApplySingle(
			writer,
			targetPool,
			wop,
		)
		if err != nil {
			return errors.WithStack(err)
		}
	}
}

func makeWop(op *pwr.SyncOp) (wsync.Operation, error) {
	switch op.Type {
	case pwr.SyncOp_BLOCK_RANGE:
		return wsync.Operation{
			Type:       wsync.OpBlockRange,
			FileIndex:  op.FileIndex,
			BlockIndex: op.BlockIndex,
			BlockSpan:  op.BlockSpan,
		}, nil
	case pwr.SyncOp_DATA:
		return wsync.Operation{
			Type: wsync.OpData,
			Data: op.Data,
		}, nil
	default:
		return wsync.Operation{}, errors.WithStack(fmt.Errorf("unknown sync op type: %s", op.Type))
	}
}

func (sp *savingPatcher) isFullFileOp(sh *pwr.SyncHeader, op *pwr.SyncOp) bool {
	// only block range ops can be full-file ops
	if op.Type != pwr.SyncOp_BLOCK_RANGE {
		return false
	}

	// and it's gotta start at 0
	if op.BlockIndex != 0 {
		return false
	}

	targetFile := sp.targetContainer.Files[op.FileIndex]
	outputFile := sp.sourceContainer.Files[sh.FileIndex]

	// and both files have gotta be the same size
	if targetFile.Size != outputFile.Size {
		return false
	}

	numOutputBlocks := pwr.ComputeNumBlocks(outputFile.Size)

	// and it's gotta, well, span the full file
	if op.BlockSpan != numOutputBlocks {
		return false
	}

	return true
}

func (sp *savingPatcher) processBsdiff(c *Checkpoint, targetPool wsync.Pool, sh *pwr.SyncHeader, bwl bowl.Bowl) error {
	var writer bowl.EntryWriter
	var old io.ReadSeeker
	var oldOffset int64
	var targetIndex int64

	if c.BsdiffCheckpoint != nil {
		var err error

		targetIndex = c.BsdiffCheckpoint.TargetIndex
		oldOffset = c.BsdiffCheckpoint.OldOffset

		old, err = targetPool.GetReadSeeker(targetIndex)
		if err != nil {
			return errors.WithStack(err)
		}

		// alrighty let's do it
		writer, err = bwl.GetWriter(sh.FileIndex)
		if err != nil {
			return errors.WithStack(err)
		}

		// FIXME: swallowed error
		defer writer.Close()

		_, err = writer.Resume(c.BsdiffCheckpoint.BowlCheckpoint)
		if err != nil {
			return errors.WithStack(err)
		}

		f := sp.sourceContainer.Files[sh.FileIndex]
		sp.consumer.Debugf("↺ Resuming bsdiff entry @ %s / %s",
			humanize.IBytes(uint64(writer.Tell())),
			humanize.IBytes(uint64(f.Size)),
		)
	} else {
		// starting from the beginning!
		var err error

		bh := &pwr.BsdiffHeader{}
		err = sp.rctx.ReadMessage(bh)
		if err != nil {
			return errors.WithStack(err)
		}

		targetIndex = bh.TargetIndex

		old, err = targetPool.GetReadSeeker(targetIndex)
		if err != nil {
			return errors.WithStack(err)
		}

		// let's patch!
		writer, err = bwl.GetWriter(sh.FileIndex)
		if err != nil {
			return errors.WithStack(err)
		}

		// FIXME: swallowed error
		defer writer.Close()

		_, err = writer.Resume(nil)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if sp.bsdiffCtx == nil {
		sp.bsdiffCtx = bsdiff.NewPatchContext()
	}

	ipc, err := sp.bsdiffCtx.NewIndividualPatchContext(
		old,
		oldOffset,
		writer,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	ctrl := &bsdiff.Control{}
	for {
		if sp.sc.ShouldSave() {
			sp.rctx.WantSave()

			messageCheckpoint := sp.rctx.PopCheckpoint()
			if messageCheckpoint != nil {
				bowlCheckpoint, err := writer.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				// aw yeah let's save this state
				checkpoint := &Checkpoint{
					SyncHeader:        sh,
					FileIndex:         sh.FileIndex,
					FileKind:          FileKindBsdiff,
					MessageCheckpoint: messageCheckpoint,
					BsdiffCheckpoint: &BsdiffCheckpoint{
						BowlCheckpoint: bowlCheckpoint,
						OldOffset:      ipc.OldOffset,
						TargetIndex:    targetIndex,
					},
				}
				action, err := sp.sc.Save(checkpoint)
				if err != nil {
					return err
				}

				switch action {
				case AfterSaveStop:
					return ErrStop
				}
			}
		}

		err = sp.rctx.ReadMessage(ctrl)
		if err != nil {
			return err
		}

		if ctrl.Eof {
			// woo!
			break
		}

		err = ipc.Apply(ctrl)
		if err != nil {
			return err
		}
	}

	// now read the sentinel syncop
	op := &pwr.SyncOp{}
	err = sp.rctx.ReadMessage(op)
	if err != nil {
		return errors.WithStack(err)
	}

	if op.Type != pwr.SyncOp_HEY_YOU_DID_IT {
		return errors.WithStack(fmt.Errorf("corrupt patch: expected sentinel SyncOp after bsdiff series, got %s", op.Type))
	}

	// now check the final size
	f := sp.sourceContainer.Files[sh.FileIndex]
	finalSize := writer.Tell()
	if finalSize != f.Size {
		err = fmt.Errorf("corrupted patch: expected '%s' to be %s (%d bytes) after patching, but it's %s (%d bytes)",
			f.Path,
			humanize.IBytes(uint64(f.Size)),
			f.Size,
			humanize.IBytes(uint64(finalSize)),
			finalSize,
		)
		return errors.WithStack(err)
	}

	// and we're done!

	return nil
}

func (sp *savingPatcher) SetSaveConsumer(sc SaveConsumer) {
	sp.sc = sc
}

func (sp *savingPatcher) GetSourceContainer() *tlc.Container {
	return sp.sourceContainer
}

func (sp *savingPatcher) GetTargetContainer() *tlc.Container {
	return sp.targetContainer
}

func (sp *savingPatcher) Progress() float64 {
	if sp.rctx == nil {
		return -2
	}

	if sp.rctx.GetSource() == nil {
		return -1
	}

	return sp.rctx.GetSource().Progress()
}
