package checker

import (
	"bytes"
	"os/exec"

	"github.com/itchio/go-brotli/enc"
	"github.com/itchio/kompress/flate"
	"github.com/itchio/kompress/gzip"

	"github.com/pkg/errors"
)

func GzipCompress(input []byte) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)
	w, err := gzip.NewWriterLevel(compressedBuf, 9)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, err = w.Write(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return compressedBuf.Bytes(), nil
}

func FlateCompress(input []byte) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)
	w, err := flate.NewWriter(compressedBuf, 9)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, err = w.Write(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return compressedBuf.Bytes(), nil
}

func Bzip2Compress(input []byte) ([]byte, error) {
	cmd := exec.Command("bzip2")
	outbuf := new(bytes.Buffer)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = outbuf

	err := cmd.Run()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return outbuf.Bytes(), nil
}

func BrotliCompress(input []byte, level int) ([]byte, error) {
	compressedBuf := new(bytes.Buffer)

	w := enc.NewBrotliWriter(compressedBuf, &enc.BrotliWriterOptions{
		Quality: level,
	})

	_, err := w.Write(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = w.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return compressedBuf.Bytes(), nil
}
