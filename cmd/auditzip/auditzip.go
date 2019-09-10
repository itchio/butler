package auditzip

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	upstreamzip "archive/zip"

	itchiozip "github.com/itchio/arkive/zip"
	"github.com/itchio/headway/united"

	"github.com/itchio/boar"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
	"github.com/pkg/errors"
)

var args = struct {
	file     *string
	upstream *bool
}{}

var doArgs = struct {
	file *string
	rfc  *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("auditzip", "Audit a zip file for common errors")
	args.file = cmd.Arg("file", ".zip file to audit").Required().String()
	args.upstream = cmd.Flag("upstream", "Use upstream zip implementation (archive/zip)").Bool()
	ctx.Register(cmd, do)

	doCmd := ctx.App.Command("mkprotozip", "Make a zip with all supported entry types")
	doArgs.file = doCmd.Arg("file", ".zip file to make").Required().String()
	doArgs.rfc = doCmd.Flag("rfc", "If set, only use STORE and DEFLATE methods").Bool()
	ctx.Register(doCmd, doMk)
}

type lzmaWriter struct {
	buf *bytes.Buffer
	w   io.Writer
}

func (lw *lzmaWriter) Write(buf []byte) (int, error) {
	n, err := lw.buf.Write(buf)
	return n, err
}

func (lw *lzmaWriter) Close() error {
	f, err := ioutil.TempFile("", "lzma-in")
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write(lw.buf.Bytes())
	if err != nil {
		return errors.WithStack(err)
	}
	f.Close()

	cmd := exec.Command("lzma", "--compress", f.Name(), "--stdout")
	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}

	var versionInfo uint16 = 5129
	var propSize uint16 = 5
	err = binary.Write(lw.w, binary.LittleEndian, versionInfo)
	if err != nil {
		return errors.WithStack(err)
	}
	err = binary.Write(lw.w, binary.LittleEndian, propSize)
	if err != nil {
		return errors.WithStack(err)
	}

	outReader := bytes.NewReader(outBuf.Bytes())

	lzmaProps := make([]byte, 5)
	_, err = io.ReadFull(outReader, lzmaProps)
	if err != nil {
		return errors.WithStack(err)
	}

	lzmaSize := make([]byte, 8)
	_, err = io.ReadFull(outReader, lzmaSize)
	if err != nil {
		return errors.WithStack(err)
	}

	var unpackSize uint64
	for i := 0; i < 8; i++ {
		b := lzmaSize[i]
		unpackSize = unpackSize | uint64(b)<<uint64(8*i)
	}

	_, err = lw.w.Write(lzmaProps)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = io.Copy(lw.w, outReader)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func doMk(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()
	ctx.Must(DoMk(consumer, *doArgs.file, *doArgs.rfc))
}

func DoMk(consumer *state.Consumer, file string, rfc bool) error {
	f, err := os.Create(file)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	itchiozip.RegisterCompressor(itchiozip.LZMA, func(s itchiozip.CompressionSettings, w io.Writer) (io.WriteCloser, error) {
		buf := new(bytes.Buffer)
		return &lzmaWriter{buf, w}, nil
	})

	w := itchiozip.NewWriter(f)
	defer w.Close()

	{
		bs := []byte("I'm a stored file")
		ew, err := w.CreateHeader(&itchiozip.FileHeader{
			Name:               "store-item",
			Method:             itchiozip.Store,
			UncompressedSize64: uint64(len(bs)),
		})
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = ew.Write(bs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	{
		bs := []byte("I'm a deflated file")
		ew, err := w.CreateHeader(&itchiozip.FileHeader{
			Name:               "deflate-item",
			Method:             itchiozip.Deflate,
			UncompressedSize64: uint64(len(bs)),
		})
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = ew.Write(bs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	if !rfc {
		bs := []byte("I'm an LZMA-compressed file")
		ew, err := w.CreateHeader(&itchiozip.FileHeader{
			Name:               "lzma-item",
			Method:             itchiozip.LZMA,
			UncompressedSize64: uint64(len(bs)),
		})
		if err != nil {
			return errors.WithStack(err)
		}

		ew.Write(bs)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func do(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()
	ctx.Must(Do(consumer, *args.file, *args.upstream))
}

func Do(consumer *state.Consumer, file string, upstream bool) error {
	f, err := eos.Open(file, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	consumer.Opf("Auditing (%s)...", stats.Name())

	var impl ZipImpl
	if upstream {
		consumer.Opf("Using upstream zip implementation")
		impl = &upstreamImpl{}
	} else {
		consumer.Opf("Using itchio/arkive zip implementation")
		impl = &itchioImpl{}
	}

	var foundErrors []string

	markError := func(path string, message string, args ...interface{}) {
		formatted := fmt.Sprintf(message, args...)
		fullMessage := fmt.Sprintf("(%s): %s", path, formatted)
		foundErrors = append(foundErrors, fullMessage)
	}

	paths := make(map[string]int)
	started := false

	err = impl.EachEntry(consumer, f, stats.Size(), func(index int, name string, nonutf8 bool, uncompressedSize int64, rc io.ReadCloser, numEntries int) error {
		if !started {
			comm.StartProgress()
			started = true
		}
		path := boar.CleanFileName(name)

		if nonutf8 {
			for _, r := range name {
				if r > 127 {
					markError(path, "Entry has non-ASCII characters but isn't encoded as utf-8")
					break
				}
			}
		}

		comm.Progress(float64(index) / float64(numEntries))
		comm.ProgressLabel(path)

		if previousIndex, ok := paths[path]; ok {
			consumer.Warnf("Duplicate path (%s) at indices (%d) and (%d)", path, index, previousIndex)
		}
		paths[path] = index

		actualSize, err := io.Copy(ioutil.Discard, rc)
		if err != nil {
			markError(path, err.Error())
			return nil
		}

		if actualSize != uncompressedSize {
			err := fmt.Errorf("Dictionary says (%s) is %s (%d bytes), but it's actually %s (%d bytes)",
				path,
				united.FormatBytes(uncompressedSize),
				uncompressedSize,
				united.FormatBytes(actualSize),
				actualSize,
			)
			return errors.WithStack(err)
		}
		return nil
	})
	comm.EndProgress()
	if err != nil {
		return errors.WithStack(err)
	}

	if len(foundErrors) > 0 {
		consumer.Infof("================================================")
		consumer.Statf("Found %d errors:", len(foundErrors))
		for _, fullMessage := range foundErrors {
			consumer.Logf(" ✖ %s", fullMessage)
		}
		consumer.Infof("================================================")
		return fmt.Errorf("Found %d errors in zip file", len(foundErrors))
	}

	consumer.Statf("Everything checks out!")

	return nil
}

// zip implementation types

type EachEntryFunc func(index int, name string, nonutf8 bool, uncompressedSize int64, rc io.ReadCloser, numEntries int) error

type ZipImpl interface {
	EachEntry(consumer *state.Consumer, r io.ReaderAt, size int64, cb EachEntryFunc) error
}

// itchio zip impl

type itchioImpl struct{}

var _ ZipImpl = (*itchioImpl)(nil)

func (a *itchioImpl) EachEntry(consumer *state.Consumer, r io.ReaderAt, size int64, cb EachEntryFunc) error {
	zr, err := itchiozip.NewReader(r, size)
	if err != nil {
		return errors.WithStack(err)
	}

	var compressedSize int64
	var uncompressedSize int64
	for _, entry := range zr.File {
		compressedSize += int64(entry.CompressedSize64)
		uncompressedSize += int64(entry.UncompressedSize64)
	}
	printExtras(consumer, size, compressedSize, uncompressedSize, zr.Comment)

	foundMethods := make(map[uint16]int)
	for _, entry := range zr.File {
		foundMethods[entry.Method] = foundMethods[entry.Method] + 1
	}
	printFoundMethods(consumer, foundMethods)

	numEntries := len(zr.File)
	for index, entry := range zr.File {
		rc, err := entry.Open()
		if err != nil {
			return errors.WithStack(err)
		}

		if entry.Method == itchiozip.LZMA {
			offset, err := entry.DataOffset()
			if err == nil {
				comm.Debugf("(%s) offset = %d, compressed data size = %d, flags = %x, reader version = %d", entry.Name, offset, entry.CompressedSize64, entry.Flags, entry.ReaderVersion)
			}
		}

		err = cb(index, entry.Name, entry.NonUTF8, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// upstream zip impl

type upstreamImpl struct{}

var _ ZipImpl = (*upstreamImpl)(nil)

func (a *upstreamImpl) EachEntry(consumer *state.Consumer, r io.ReaderAt, size int64, cb EachEntryFunc) error {
	zr, err := upstreamzip.NewReader(r, size)
	if err != nil {
		return errors.WithStack(err)
	}

	var compressedSize int64
	var uncompressedSize int64
	for _, entry := range zr.File {
		compressedSize += int64(entry.CompressedSize64)
		uncompressedSize += int64(entry.UncompressedSize64)
	}
	printExtras(consumer, size, compressedSize, uncompressedSize, zr.Comment)

	foundMethods := make(map[uint16]int)
	for _, entry := range zr.File {
		foundMethods[entry.Method] = foundMethods[entry.Method] + 1
	}
	printFoundMethods(consumer, foundMethods)

	numEntries := len(zr.File)
	for index, entry := range zr.File {
		rc, err := entry.Open()
		if err != nil {
			return errors.WithStack(err)
		}

		err = cb(index, entry.Name, entry.NonUTF8, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// utils

func printExtras(consumer *state.Consumer, size int64, compressedSize int64, uncompressedSize int64, comment string) {
	consumer.Infof("Comment: (%s)", comment)
	consumer.Infof("Sizes: ")
	consumer.Infof(" → Archive size      : %s (%d bytes)", united.FormatBytes(size), size)
	consumer.Infof(" → Sum (compressed)  : %s (%d bytes)", united.FormatBytes(compressedSize), compressedSize)
	consumer.Infof(" → Sum (uncompressed): %s (%d bytes)", united.FormatBytes(uncompressedSize), uncompressedSize)
	if compressedSize > uncompressedSize {
		consumer.Warnf("Compressed size is larger than uncompressed, that's suspicious.")
	}
}

func printFoundMethods(consumer *state.Consumer, foundMethods map[uint16]int) {
	consumer.Infof("Entries: ")
	for method, count := range foundMethods {
		switch method {
		case itchiozip.Store:
			consumer.Infof(" → %d STORE entries", count)
		case itchiozip.Deflate:
			consumer.Infof(" → %d DEFLATE entries", count)
		case itchiozip.LZMA:
			consumer.Infof(" → %d LZMA entries", count)
		default:
			consumer.Infof(" → %d entries with unknown method (%d)", count, method)
		}
	}
}
