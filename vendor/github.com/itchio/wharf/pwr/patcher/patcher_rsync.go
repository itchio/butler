package patcher

import (
	"fmt"
	"sync"

	"github.com/andreyvit/diff"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

func (sp *savingPatcher) processRsync(c *Checkpoint, targetPool wsync.Pool, sh *pwr.SyncHeader, bwl bowl.Bowl) (err error) {
	var op *pwr.SyncOp

	var writer bowl.EntryWriter
	var closeWriterOnce sync.Once

	if c.RsyncCheckpoint != nil {
		// alrighty let's do it
		writer, err = bwl.GetWriter(sh.FileIndex)
		if err != nil {
			return err
		}

		_, err = writer.Resume(c.RsyncCheckpoint.WriterCheckpoint)
		if err != nil {
			return err
		}

		defer closeWriterOnce.Do(func() {
			cerr := writer.Close()
			if err == nil && cerr != nil {
				err = cerr
			}
		})

		f := sp.sourceContainer.Files[sh.FileIndex]
		sp.consumer.Debugf("↺ Resuming rsync entry @ %s / %s",
			progress.FormatBytes(writer.Tell()),
			progress.FormatBytes(f.Size),
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
			oldName := sp.targetContainer.Files[op.FileIndex].Path
			newName := sp.sourceContainer.Files[sh.FileIndex].Path
			sp.consumer.Debugf("Transpose: %s", diff.CharacterDiff(oldName, newName))

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

		_, err = writer.Resume(nil)
		if err != nil {
			return errors.WithStack(err)
		}

		defer closeWriterOnce.Do(func() {
			cerr := writer.Close()
			if err == nil && cerr != nil {
				err = cerr
			}
		})
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

		var wop wsync.Operation
		wop, err = makeWop(op)
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
				bowlCheckpoint, err := bwl.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				writerCheckpoint, err := writer.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				checkpoint := &Checkpoint{
					SyncHeader:        sh,
					FileIndex:         sh.FileIndex,
					FileKind:          FileKindRsync,
					MessageCheckpoint: messageCheckpoint,
					BowlCheckpoint:    bowlCheckpoint,
					RsyncCheckpoint: &RsyncCheckpoint{
						WriterCheckpoint: writerCheckpoint,
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

			// let's not forget to finalize the file
			err = writer.Finalize()
			if err != nil {
				return err
			}

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
