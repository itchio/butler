package checker

import (
	"bytes"

	"github.com/dsnet/compress/bzip2"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/kompress/gzip"

	"github.com/go-errors/errors"
)

func GzipCompress(input []byte) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)
	w, err := gzip.NewWriterLevel(compressedBuf, 9)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	_, err = w.Write(input)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return compressedBuf.Bytes(), nil
}

func FlateCompress(input []byte) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)
	w, err := flate.NewWriter(compressedBuf, 9)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	_, err = w.Write(input)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return compressedBuf.Bytes(), nil
}

func Bzip2Compress(input []byte) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)
	w, err := bzip2.NewWriter(compressedBuf, &bzip2.WriterConfig{Level: 2})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	_, err = w.Write(input)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return compressedBuf.Bytes(), nil
}
