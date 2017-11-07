package pwr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"hash/crc32"

	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/wharf/crc32c"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wire"
)

type ZipIndexerContext struct {
	Compression *CompressionSettings
	Consumer    *state.Consumer

	TotalSegments         int64
	TotalCompressedSize   int64
	TotalUncompressedSize int64
	LargestSegmentSize    int64
}

type Segment struct {
	c    *flate.Checkpoint
	hash uint32
}

func (ic *ZipIndexerContext) Index(reader io.ReaderAt, size int64, writer io.Writer) error {
	wctx := wire.NewWriteContext(writer)

	err := wctx.WriteMagic(ZipIndexMagic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	compression := ic.Compression
	if compression == nil {
		compression = defaultIndexCompressionSettings()
	}

	wctx, err = CompressWire(wctx, compression)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	zih := &ZipIndexHeader{
		Compression: compression,
	}

	err = wctx.WriteMessage(zih)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	buf := make([]byte, 64*1024)
	hasher := crc32.New(crc32c.Table)

	for _, file := range zr.File {
		ic.TotalCompressedSize += int64(file.CompressedSize64)
		ic.TotalUncompressedSize += int64(file.UncompressedSize64)
	}

	doneCompressedSize := int64(0)

	for fileIndex, file := range zr.File {
		// skip dirs and symlinks
		if file.FileInfo().IsDir() {
			continue
		}

		if file.FileInfo().Mode()&os.ModeSymlink > 0 {
			continue
		}

		if file.Method != zip.Deflate {
			// we only support deflate:
			//   - LZMA is hard to index
			//   - PPMd, WavPack etc. we don't decompress
			//   - Store doesn't need any indexing
			continue
		}

		hasher.Reset()

		dataOffset, err := file.DataOffset()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		sr := io.NewSectionReader(reader, dataOffset, int64(file.CompressedSize64))

		var segments []Segment

		fr := flate.NewSaverReader(sr)
		var readBytes int64
		var threshold int64 = 4 * 1024 * 1024

		ic.Consumer.ProgressLabel(file.Name)

		lastROffset := int64(0)
		var lastCheckpoint *flate.Checkpoint

		for {
			n, err := fr.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				} else if err == flate.ReadyToSaveError {
					c, err := fr.Save()
					if err != nil {
						return errors.Wrap(err, 0)
					}

					locallyDoneCompressedSize := doneCompressedSize + c.Roffset
					ic.Consumer.Progress(float64(locallyDoneCompressedSize) / float64(ic.TotalCompressedSize))

					_, err = sr.Seek(0, io.SeekStart)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					fr, err = c.Resume(sr)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					hash := hasher.Sum32()

					segments = append(segments, Segment{
						c:    lastCheckpoint,
						hash: hash,
					})

					hasher.Reset()

					segmentSize := c.Roffset - lastROffset
					if segmentSize > ic.LargestSegmentSize {
						ic.LargestSegmentSize = segmentSize
					}
					lastROffset = c.Roffset

					ic.Consumer.ProgressLabel(fmt.Sprintf("%s [%d]", file.Name, len(segments)))
					lastCheckpoint = c
				} else {
					return errors.Wrap(err, 0)
				}
			}

			hasher.Write(buf[:n])

			readBytes += int64(n)
			if readBytes > threshold {
				fr.WantSave()
				readBytes = 0
			}
		}

		// add last checkpoint (can be the only one)
		if lastROffset < int64(file.CompressedSize64) {
			hash := hasher.Sum32()
			segments = append(segments, Segment{
				c:    lastCheckpoint,
				hash: hash,
			})
		}

		if len(segments) > 0 {
			err = wctx.WriteMessage(&ZipIndexEntryStart{
				FileOffset: int64(fileIndex),
				NumEntries: int64(len(segments)),
				Path:       filepath.ToSlash(filepath.Clean(filepath.ToSlash(file.Name))),
			})
			if err != nil {
				return errors.Wrap(err, 0)
			}

			for i, s := range segments {
				c := s.c
				if c == nil {
					if i != 0 {
						return errors.Wrap(errors.New("Internal error: segment has null checkpoint, but isn't first segment"), 0)
					}
					wctx.WriteMessage(&ZipIndexEntry{
						ROffset: 0,
						WOffset: 0,
						Hash:    s.hash,
					})
				} else {
					wctx.WriteMessage(&ZipIndexEntry{
						ROffset: c.Roffset,
						WOffset: c.Woffset,
						Hist:    c.Hist,
						Nb:      uint32(c.Nb),
						B:       c.B,
						Hash:    s.hash,
					})
				}
				if err != nil {
					return errors.Wrap(err, 0)
				}

				ic.TotalSegments++
			}
		}

		doneCompressedSize += int64(file.CompressedSize64)
		ic.Consumer.Progress(float64(doneCompressedSize) / float64(ic.TotalCompressedSize))
	}

	return nil
}

func defaultIndexCompressionSettings() *CompressionSettings {
	return &CompressionSettings{
		Algorithm: CompressionAlgorithm_ZSTD,
		Quality:   9,
	}
}
