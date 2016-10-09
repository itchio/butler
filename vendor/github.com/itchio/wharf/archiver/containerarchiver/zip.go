package containerarchiver

import (
	"archive/zip"
	"io"
	"os"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/archiver"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

func CompressZip(archiveWriter io.Writer, container *tlc.Container, pool wsync.Pool, consumer *state.Consumer) (*archiver.CompressResult, error) {
	var err error
	var uncompressedSize int64
	var compressedSize int64

	archiveCounter := counter.NewWriter(archiveWriter)

	zipWriter := zip.NewWriter(archiveCounter)
	defer zipWriter.Close()
	defer func() {
		if zipWriter != nil {
			if zErr := zipWriter.Close(); err == nil && zErr != nil {
				err = errors.Wrap(zErr, 1)
			}
		}
	}()

	for _, dir := range container.Dirs {
		fh := zip.FileHeader{
			Name: dir.Path + "/",
		}
		fh.SetMode(os.FileMode(dir.Mode))
		fh.SetModTime(time.Now())

		_, hErr := zipWriter.CreateHeader(&fh)
		if hErr != nil {
			return nil, errors.Wrap(hErr, 1)
		}
	}

	for fileIndex, file := range container.Files {
		fh := zip.FileHeader{
			Name:               file.Path,
			UncompressedSize64: uint64(file.Size),
			Method:             zip.Deflate,
		}
		fh.SetMode(os.FileMode(file.Mode))
		fh.SetModTime(time.Now())

		entryWriter, eErr := zipWriter.CreateHeader(&fh)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		entryReader, eErr := pool.GetReader(int64(fileIndex))
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		copiedBytes, eErr := io.Copy(entryWriter, entryReader)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		uncompressedSize += copiedBytes
	}

	for _, symlink := range container.Symlinks {
		fh := zip.FileHeader{
			Name: symlink.Path,
		}
		fh.SetMode(os.FileMode(symlink.Mode))

		entryWriter, eErr := zipWriter.CreateHeader(&fh)
		if eErr != nil {
			return nil, errors.Wrap(eErr, 1)
		}

		entryWriter.Write([]byte(symlink.Dest))
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	zipWriter = nil

	compressedSize = archiveCounter.Count()

	return &archiver.CompressResult{
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
	}, nil
}
