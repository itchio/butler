package blockpool

import (
	"bytes"
	"io"

	"github.com/Datadog/zstd"
	"github.com/pkg/errors"
)

/////////////////////////////
// Compressor
/////////////////////////////

// A Compressor compresses blocks with ztsd-q9
type Compressor struct {
	compressedBuf []byte
}

func (c *Compressor) Clone() *Compressor {
	return &Compressor{}
}

// Store first compresses the data, then stores it into the underlying sink
func (c *Compressor) Compress(writer io.Writer, in []byte) error {
	if c.compressedBuf == nil {
		c.compressedBuf = make([]byte, BigBlockSize*2)
	}

	compressedBuf, err := zstd.CompressLevel(c.compressedBuf, in, 9)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = writer.Write(compressedBuf)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

/////////////////////////////
// Decompressor
/////////////////////////////

// A Decompressor decompresses zstd-compressed blocks
type Decompressor struct {
	buffer *bytes.Buffer
}

func (d *Decompressor) Clone() *Decompressor {
	return &Decompressor{}
}

// Fetch first fetches from the underlying source, then decompresses
func (d *Decompressor) Decompress(out []byte, reader io.Reader) (int, error) {
	if d.buffer == nil {
		d.buffer = new(bytes.Buffer)
		d.buffer.Grow(int(BigBlockSize * 2))
	}

	d.buffer.Reset()

	compressedBytes, err := io.Copy(d.buffer, reader)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	decompressedBuf, err := zstd.Decompress(out, d.buffer.Bytes()[:compressedBytes])
	if err != nil {
		return 0, errors.WithStack(err)
	}

	return len(decompressedBuf), nil
}
