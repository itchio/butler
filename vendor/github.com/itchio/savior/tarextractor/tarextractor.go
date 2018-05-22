package tarextractor

import (
	"encoding/gob"
	"io"
	"os"

	"github.com/itchio/arkive/tar"
	"github.com/itchio/httpkit/progress"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type tarExtractor struct {
	source savior.Source

	saveConsumer savior.SaveConsumer
	consumer     *state.Consumer
}

type TarExtractorState struct {
	Result        *savior.ExtractorResult
	TarCheckpoint *tar.Checkpoint
}

var _ savior.Extractor = (*tarExtractor)(nil)

func New(source savior.Source) savior.Extractor {
	return &tarExtractor{
		source:       source,
		saveConsumer: savior.NopSaveConsumer(),
		consumer:     savior.NopConsumer(),
	}
}

func (te *tarExtractor) SetSaveConsumer(saveConsumer savior.SaveConsumer) {
	te.saveConsumer = saveConsumer
}

func (te *tarExtractor) SetConsumer(consumer *state.Consumer) {
	te.consumer = consumer
}

func (te *tarExtractor) Resume(checkpoint *savior.ExtractorCheckpoint, sink savior.Sink) (*savior.ExtractorResult, error) {
	var sr tar.SaverReader
	var state *TarExtractorState

	if checkpoint != nil {
		if stateCheckpoint, ok := checkpoint.Data.(*TarExtractorState); ok {
			if stateCheckpoint.Result != nil && stateCheckpoint.TarCheckpoint != nil {
				te.consumer.Infof("↻ Resuming @ %.1f%%", checkpoint.Progress*100)

				if checkpoint.SourceCheckpoint != nil {
					savior.Debugf("tarextractor: resuming source from %d", checkpoint.SourceCheckpoint.Offset)
				}
				offset, err := te.source.Resume(checkpoint.SourceCheckpoint)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				tarCheckpoint := stateCheckpoint.TarCheckpoint
				if offset < tarCheckpoint.Roffset {
					delta := tarCheckpoint.Roffset - offset
					savior.Debugf("tarextractor: discarding %d bytes to align source and tar checkpoint", delta)
					savior.Debugf("tarextractor: source was at %d, tar checkpoint was at %d", offset, tarCheckpoint.Roffset)
					err = savior.DiscardByRead(te.source, delta)
					if err != nil {
						return nil, errors.WithStack(err)
					}
				}

				sr, err = tarCheckpoint.Resume(te.source)
				if err != nil {
					return nil, errors.WithStack(err)
				}

				state = stateCheckpoint
			}
		}
	}

	if sr == nil {
		te.consumer.Infof("→ Starting fresh extraction")

		state = &TarExtractorState{
			Result: &savior.ExtractorResult{
				Entries: []*savior.Entry{},
			},
		}

		_, err := te.source.Resume(nil)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		checkpoint = &savior.ExtractorCheckpoint{
			EntryIndex: 0,
		}

		sr, err = tar.NewSaverReader(te.source)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	var stopError error

	// allocate a copy buffer once
	copier := savior.NewCopier(te.saveConsumer)

	var entry *savior.Entry
	te.source.SetSourceSaveConsumer(&savior.CallbackSourceSaveConsumer{
		OnSave: func(sourceCheckpoint *savior.SourceCheckpoint) error {
			if entry == nil {
				// if entry is nil here, then our source emitted a checkpoint
				// during a call to `Next()`
				return nil
			}

			savior.Debugf("tarextractor: making checkpoint at entry %d", checkpoint.EntryIndex)

			tarCheckpoint, err := sr.Save()
			if err != nil {
				return errors.WithStack(err)
			}
			savior.Debugf("tarextractor: at checkpoint, tar read offset is %s", progress.FormatBytes(tarCheckpoint.Roffset))

			state.TarCheckpoint = tarCheckpoint

			checkpoint.SourceCheckpoint = sourceCheckpoint
			checkpoint.Data = state
			checkpoint.Progress = te.source.Progress()

			// FIXME: we're not syncing the writer here - but we should

			action, err := te.saveConsumer.Save(checkpoint)
			if err != nil {
				return errors.WithStack(err)
			}
			if action == savior.AfterSaveStop {
				copier.Stop()
				stopError = savior.ErrStop
			}
			return nil
		},
	})

	entryIndex := checkpoint.EntryIndex
	for stopError == nil {
		err := func() error {
			entry = nil

			checkpoint.EntryIndex = entryIndex
			entryIndex++

			if checkpoint.Entry == nil {
				hdr, err := sr.Next()
				if err != nil {
					if err == io.EOF {
						// we done!
						stopError = io.EOF
						return nil
					}
					return errors.WithStack(err)
				}

				entry := &savior.Entry{
					CanonicalPath:    hdr.Name,
					UncompressedSize: hdr.Size,
					Mode:             os.FileMode(hdr.Mode),
				}

				switch hdr.Typeflag {
				case tar.TypeDir:
					entry.Kind = savior.EntryKindDir
				case tar.TypeSymlink:
					entry.Kind = savior.EntryKindSymlink
					entry.Linkname = hdr.Linkname
				case tar.TypeReg:
					entry.Kind = savior.EntryKindFile
				default:
					// let's just ignore that one..
					return nil
				}
				checkpoint.Entry = entry
			}
			entry = checkpoint.Entry

			te.consumer.Debugf("→ %s", entry)

			switch entry.Kind {
			case savior.EntryKindDir:
				savior.Debugf(`tar: extracting dir %s`, entry.CanonicalPath)
				err := sink.Mkdir(entry)
				if err != nil {
					return errors.WithStack(err)
				}
			case savior.EntryKindSymlink:
				savior.Debugf(`tar: extracting symlink %s`, entry.CanonicalPath)
				err := sink.Symlink(entry, entry.Linkname)
				if err != nil {
					return errors.WithStack(err)
				}
			case savior.EntryKindFile:
				savior.Debugf(`tar: extracting file %s`, entry.CanonicalPath)
				w, err := sink.GetWriter(entry)
				if err != nil {
					return errors.WithStack(err)
				}
				defer w.Close()

				err = copier.Do(&savior.CopyParams{
					Dst:   w,
					Src:   sr,
					Entry: entry,

					Savable: te.source,

					EmitProgress: func() {
						te.consumer.Progress(te.source.Progress())
					},
				})
				if err != nil {
					return errors.WithStack(err)
				}

				state.Result.Entries = append(state.Result.Entries, entry)
				te.consumer.Progress(te.source.Progress())
			}

			checkpoint.Entry = nil
			checkpoint.SourceCheckpoint = nil
			checkpoint.Data = nil

			return nil
		}()
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if stopError != nil {
		if stopError == io.EOF {
			// success!
		} else {
			return nil, stopError
		}
	}

	te.consumer.Statf("Extracted %s", state.Result.Stats())
	return state.Result, nil
}

func (te *tarExtractor) Features() savior.ExtractorFeatures {
	sf := te.source.Features()

	// tar's resume support depends on the underlying source
	var resumeSupport savior.ResumeSupport
	switch sf.ResumeSupport {
	case savior.ResumeSupportBlock:
		resumeSupport = savior.ResumeSupportBlock
	default:
		resumeSupport = savior.ResumeSupportNone
	}

	// but we cannot preallocate or grant random access.
	return savior.ExtractorFeatures{
		Name:           "tar",
		ResumeSupport:  resumeSupport,
		Preallocate:    false,
		RandomAccess:   false,
		SourceFeatures: &sf,
	}
}

func init() {
	gob.Register(&TarExtractorState{})
	gob.Register(&tar.Checkpoint{})
}
