package tarextractor

import (
	"encoding/gob"
	"io"

	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/tar"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
)

type tarExtractor struct {
	source savior.Source
	sink   savior.Sink

	saveConsumer savior.SaveConsumer
	consumer     *state.Consumer
}

type TarExtractorState struct {
	Result        *savior.ExtractorResult
	TarCheckpoint *tar.Checkpoint
}

var _ savior.Extractor = (*tarExtractor)(nil)

func New(source savior.Source, sink savior.Sink) savior.Extractor {
	return &tarExtractor{
		source:       source,
		sink:         sink,
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

func (te *tarExtractor) Resume(checkpoint *savior.ExtractorCheckpoint) (*savior.ExtractorResult, error) {
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
					return nil, errors.Wrap(err, 0)
				}

				tarCheckpoint := stateCheckpoint.TarCheckpoint
				if offset < tarCheckpoint.Roffset {
					delta := tarCheckpoint.Roffset - offset
					savior.Debugf("tarextractor: discarding %d bytes to align source and tar checkpoint", delta)
					savior.Debugf("tarextractor: source was at %d, tar checkpoint was at %d", offset, tarCheckpoint.Roffset)
					err = savior.DiscardByRead(te.source, delta)
					if err != nil {
						return nil, errors.Wrap(err, 0)
					}
				}

				sr, err = tarCheckpoint.Resume(te.source)
				if err != nil {
					return nil, errors.Wrap(err, 0)
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
			return nil, errors.Wrap(err, 0)
		}

		checkpoint = &savior.ExtractorCheckpoint{
			EntryIndex: 0,
		}

		sr, err = tar.NewSaverReader(te.source)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	stop := false
	var stopErr error
	entryIndex := checkpoint.EntryIndex
	for {
		if stop {
			if stopErr == nil {
				te.consumer.Statf("Extracted %s", state.Result.Stats())
				return state.Result, nil
			}
			return nil, stopErr
		}

		err := func() error {
			checkpoint.EntryIndex = entryIndex
			entryIndex++

			if checkpoint.Entry == nil {
				hdr, err := sr.Next()
				if err != nil {
					if err == io.EOF {
						// we done!
						stop = true
						return nil
					}
					return errors.Wrap(err, 0)
				}

				entry := &savior.Entry{
					CanonicalPath:    hdr.Name,
					UncompressedSize: hdr.Size,
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
			entry := checkpoint.Entry

			te.consumer.Debugf("→ %s", entry)

			switch entry.Kind {
			case savior.EntryKindDir:
				savior.Debugf(`tar: extracting dir %s`, entry.CanonicalPath)
				err := te.sink.Mkdir(entry)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			case savior.EntryKindSymlink:
				savior.Debugf(`tar: extracting symlink %s`, entry.CanonicalPath)
				err := te.sink.Symlink(entry, entry.Linkname)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			case savior.EntryKindFile:
				savior.Debugf(`tar: extracting file %s`, entry.CanonicalPath)
				w, err := te.sink.GetWriter(entry)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				defer w.Close()

				copyRes, err := savior.CopyWithSaver(&savior.CopyParams{
					Dst:   w,
					Src:   sr,
					Entry: entry,

					SaveConsumer: te.saveConsumer,
					MakeCheckpoint: func() (*savior.ExtractorCheckpoint, error) {
						savior.Debugf("tarextractor: making checkpoint at entry %d", checkpoint.EntryIndex)
						sourceCheckpoint, err := te.source.Save()
						if err != nil {
							return nil, errors.Wrap(err, 0)
						}
						if sourceCheckpoint != nil {
							savior.Debugf("tarextractor: at checkpoint, source is at %s", humanize.IBytes(uint64(sourceCheckpoint.Offset)))
						}

						tarCheckpoint, err := sr.Save()
						if err != nil {
							return nil, errors.Wrap(err, 0)
						}
						savior.Debugf("tarextractor: at checkpoint, tar read offset is %s", humanize.IBytes(uint64(tarCheckpoint.Roffset)))

						state.TarCheckpoint = tarCheckpoint

						checkpoint.SourceCheckpoint = sourceCheckpoint
						checkpoint.Data = state
						checkpoint.Progress = te.source.Progress()

						return checkpoint, nil
					},

					EmitProgress: func() {
						te.consumer.Progress(te.source.Progress())
					},
				})
				if err != nil {
					return errors.Wrap(err, 0)
				}

				if copyRes.Action == savior.AfterSaveStop {
					stop = true
					stopErr = savior.StopErr
					return nil
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
			return nil, errors.Wrap(err, 0)
		}
	}
}

func (te *tarExtractor) Features() savior.ExtractorFeatures {
	// tar has great resume support but cannot preallocate or grant random access.
	return savior.ExtractorFeatures{
		Name:          "tar",
		ResumeSupport: savior.ResumeSupportBlock,
		Preallocate:   false,
		RandomAccess:  false,
	}
}

func init() {
	gob.Register(&TarExtractorState{})
	gob.Register(&tar.Checkpoint{})
}
