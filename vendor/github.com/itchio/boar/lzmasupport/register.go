package lzmasupport

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/itchio/arkive/zip"
	"github.com/itchio/lzma"
	"github.com/pkg/errors"
)

func init() {
	zip.RegisterDecompressor(zip.LZMA, lzmaDecompressor)
}

func lzmaDecompressor(r io.Reader, f *zip.File) io.ReadCloser {
	return &initReadCloser{
		init: func() (io.ReadCloser, error) {
			// LZMA-compressed zip entries have:
			// - LZMA version info (2 bytes)
			// - LZMA properties block size (2 bytes)
			// - LZMA properties (variable size)

			var versionInfo uint16
			err := binary.Read(r, binary.LittleEndian, &versionInfo)
			if err != nil {
				return nil, errors.Wrap(err, "while reading LZMA zip entry version info")
			}

			var propSize uint16
			err = binary.Read(r, binary.LittleEndian, &propSize)
			if err != nil {
				return nil, errors.Wrap(err, "while reading LZMA zip entry properties size")
			}

			lzmaProps := make([]byte, propSize)
			_, err = io.ReadFull(r, lzmaProps)
			if err != nil {
				return nil, errors.Wrap(err, "while reading LZMA zip entry properties")
			}

			lzmaSize := make([]byte, 8)
			for i := uint32(0); i < 8; i++ {
				lzmaSize[i] = byte(f.UncompressedSize64 >> (8 * i))
			}

			// "Classic LZMA" headers are:
			// LZMA properties (5 bytes)
			// Uncompressed size (8 bytes)
			// We reproduce it here with a concatReader
			cr := &concatReader{
				readers: []io.Reader{
					bytes.NewReader(lzmaProps),
					bytes.NewReader(lzmaSize),
					r,
				},
			}

			return lzma.NewReader(cr), nil
		},
	}
}
