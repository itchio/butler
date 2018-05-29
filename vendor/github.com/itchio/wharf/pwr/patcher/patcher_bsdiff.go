package patcher

import (
	"fmt"
	"io"
	"sync"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/wharf/bsdiff"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/pwr/bowl"
	"github.com/itchio/wharf/wsync"
	"github.com/pkg/errors"
)

func (sp *savingPatcher) processBsdiff(c *Checkpoint, targetPool wsync.Pool, sh *pwr.SyncHeader, bwl bowl.Bowl) (err error) {
	var writer bowl.EntryWriter
	var closeWriterOnce sync.Once

	var old io.ReadSeeker
	var oldOffset int64
	var targetIndex int64

	if c.BsdiffCheckpoint != nil {
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

		defer closeWriterOnce.Do(func() {
			cerr := writer.Close()
			if err == nil && cerr != nil {
				err = cerr
			}
		})

		_, err = writer.Resume(c.BsdiffCheckpoint.WriterCheckpoint)
		if err != nil {
			return errors.WithStack(err)
		}

		f := sp.sourceContainer.Files[sh.FileIndex]
		sp.consumer.Debugf("↺ Resuming bsdiff entry @ %s / %s",
			progress.FormatBytes(writer.Tell()),
			progress.FormatBytes(f.Size),
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
				bowlCheckpoint, err := bwl.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				writerCheckpoint, err := writer.Save()
				if err != nil {
					return errors.WithStack(err)
				}

				// aw yeah let's save this state
				checkpoint := &Checkpoint{
					SyncHeader:        sh,
					FileIndex:         sh.FileIndex,
					FileKind:          FileKindBsdiff,
					MessageCheckpoint: messageCheckpoint,
					BowlCheckpoint:    bowlCheckpoint,
					BsdiffCheckpoint: &BsdiffCheckpoint{
						WriterCheckpoint: writerCheckpoint,
						OldOffset:        ipc.OldOffset,
						TargetIndex:      targetIndex,
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
			progress.FormatBytes(f.Size),
			f.Size,
			progress.FormatBytes(finalSize),
			finalSize,
		)
		return errors.WithStack(err)
	}

	// and finalize the writer
	err = writer.Finalize()
	if err != nil {
		return err
	}

	// and we're done!

	return nil
}
