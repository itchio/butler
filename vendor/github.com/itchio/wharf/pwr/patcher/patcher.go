package patcher

import (
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

func (sp *savingPatcher) Resume(c *Checkpoint, targetPool wsync.Pool, bwl bowl.Bowl) error {
	// we're going to open some readers while patching, and no matter what happens
	// we want to have it closed at the end (if we error out early or if we complete successfully)
	defer targetPool.Close()

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

	err := bwl.Resume(c.BowlCheckpoint)
	if err != nil {
		return errors.WithMessage(err, "while resuming bowl")
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

		err := sp.processFile(c, targetPool, sh, bwl)
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
	sp.consumer.ProgressLabel(sp.sourceContainer.Files[sh.FileIndex].Path)

	switch c.FileKind {
	case FileKindRsync:
		return sp.processRsync(c, targetPool, sh, bwl)
	case FileKindBsdiff:
		return sp.processBsdiff(c, targetPool, sh, bwl)
	default:
		return errors.Errorf("unknown file kind %d", sh.Type)
	}
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
