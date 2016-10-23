package pwr

import (
	"bytes"
	"io"
	"testing"

	"github.com/alecthomas/assert"
	"github.com/itchio/wharf/wire"
)

type fakeCompressor struct {
	called  bool
	quality int32
}

var _ Compressor = (*fakeCompressor)(nil)

func (fc *fakeCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	fc.called = true
	fc.quality = quality
	return writer, nil
}

type fakeDecompressor struct {
	called bool
}

var _ Decompressor = (*fakeDecompressor)(nil)

func (fd *fakeDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	fd.called = true
	return reader, nil
}

func Test_Compression(t *testing.T) {
	fc := &fakeCompressor{}
	RegisterCompressor(CompressionAlgorithm_GZIP, fc)

	fd := &fakeDecompressor{}
	RegisterDecompressor(CompressionAlgorithm_GZIP, fd)

	assert.EqualValues(t, false, fc.called)

	buf := new(bytes.Buffer)
	wc := wire.NewWriteContext(buf)
	_, err := CompressWire(wc, &CompressionSettings{
		Algorithm: CompressionAlgorithm_BROTLI,
		Quality:   3,
	})
	assert.NotNil(t, err)

	cwc, err := CompressWire(wc, &CompressionSettings{
		Algorithm: CompressionAlgorithm_GZIP,
		Quality:   3,
	})
	assert.NoError(t, err)

	assert.EqualValues(t, true, fc.called)
	assert.EqualValues(t, 3, fc.quality)

	assert.NoError(t, cwc.WriteMessage(&SyncHeader{
		FileIndex: 672,
	}))

	rc := wire.NewReadContext(bytes.NewReader(buf.Bytes()))

	sh := &SyncHeader{}
	assert.NoError(t, rc.ReadMessage(sh))

	assert.EqualValues(t, 672, sh.FileIndex)
	assert.NotNil(t, rc.ReadMessage(sh))
}
