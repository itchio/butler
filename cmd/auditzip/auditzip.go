package auditzip

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	upstreamzip "archive/zip"

	itchiozip "github.com/itchio/arkive/zip"

	humanize "github.com/dustin/go-humanize"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	file     *string
	upstream *bool
}{}

var doArgs = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("auditzip", "Audit a zip file for common errors")
	args.file = cmd.Arg("file", ".zip file to audit").Required().String()
	args.upstream = cmd.Flag("upstream", "Use upstream zip implementation (archive/zip)").Bool()
	ctx.Register(cmd, do)

	doCmd := ctx.App.Command("mkprotozip", "Make a zip with all supported entry types")
	doArgs.file = doCmd.Arg("file", ".zip file to make").Required().String()
	ctx.Register(doCmd, doMk)
}

type waitOnClose struct {
	pw  io.WriteCloser
	cmd *exec.Cmd
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
		return errors.Wrap(err, 0)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write(lw.buf.Bytes())
	if err != nil {
		return errors.Wrap(err, 0)
	}
	f.Close()

	cmd := exec.Command("lzma", "--compress", f.Name(), "--stdout")
	outBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var versionInfo uint16 = 5129
	var propSize uint16 = 5
	err = binary.Write(lw.w, binary.LittleEndian, versionInfo)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	err = binary.Write(lw.w, binary.LittleEndian, propSize)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	outReader := bytes.NewReader(outBuf.Bytes())

	lzmaProps := make([]byte, 5)
	_, err = io.ReadFull(outReader, lzmaProps)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	log.Printf("lzmaProps = %x", lzmaProps)

	lzmaSize := make([]byte, 8)
	_, err = io.ReadFull(outReader, lzmaSize)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var unpackSize uint64
	for i := 0; i < 8; i++ {
		b := lzmaSize[i]
		unpackSize = unpackSize | uint64(b)<<uint64(8*i)
	}
	log.Printf("lzmaSize = %x (%d)", lzmaSize, unpackSize)

	_, err = lw.w.Write(lzmaProps)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	_, err = io.Copy(lw.w, outReader)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func doMk(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()
	ctx.Must(DoMk(consumer, *doArgs.file))
}

func DoMk(consumer *state.Consumer, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	itchiozip.RegisterCompressor(itchiozip.LZMA, func(w io.Writer) (io.WriteCloser, error) {
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
			return errors.Wrap(err, 0)
		}

		_, err = ew.Write(bs)
		if err != nil {
			return errors.Wrap(err, 0)
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
			return errors.Wrap(err, 0)
		}

		_, err = ew.Write(bs)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	{
		bs := []byte("I'm an LZMA-compressed file")
		ew, err := w.CreateHeader(&itchiozip.FileHeader{
			Name:               "lzma-item",
			Method:             itchiozip.LZMA,
			UncompressedSize64: uint64(len(bs)),
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		ew.Write(bs)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func do(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()
	ctx.Must(Do(consumer, *args.file))
}

func Do(consumer *state.Consumer, file string) error {
	f, err := eos.Open(file)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Auditing (%s)...", stats.Name())

	var impl ZipImpl
	if *args.upstream {
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
		consumer.Errorf(fullMessage)
		foundErrors = append(foundErrors, fullMessage)
	}

	paths := make(map[string]int)
	started := false

	err = impl.EachEntry(consumer, f, stats.Size(), func(index int, name string, uncompressedSize int64, rc io.ReadCloser, numEntries int) error {
		if !started {
			comm.StartProgress()
			started = true
		}
		path := archive.CleanFileName(name)

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
				humanize.IBytes(uint64(uncompressedSize)),
				uncompressedSize,
				humanize.IBytes(uint64(actualSize)),
				actualSize,
			)
			return errors.Wrap(err, 0)
		}
		return nil
	})
	comm.EndProgress()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if len(foundErrors) > 0 {
		consumer.Statf("Found %d errors, see above", len(foundErrors))
		return fmt.Errorf("Found %d errors in zip file", len(foundErrors))
	}

	consumer.Statf("Everything checks out!")

	return nil
}

// zip implementation types

type EachEntryFunc func(index int, name string, uncompressedSize int64, rc io.ReadCloser, numEntries int) error

type ZipImpl interface {
	EachEntry(consumer *state.Consumer, r io.ReaderAt, size int64, cb EachEntryFunc) error
}

// itchio zip impl

type itchioImpl struct{}

var _ ZipImpl = (*itchioImpl)(nil)

func (a *itchioImpl) EachEntry(consumer *state.Consumer, r io.ReaderAt, size int64, cb EachEntryFunc) error {
	zr, err := itchiozip.NewReader(r, size)
	if err != nil {
		return errors.Wrap(err, 0)
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
			return errors.Wrap(err, 0)
		}

		if entry.Method == itchiozip.LZMA {
			offset, err := entry.DataOffset()
			if err == nil {
				comm.Debugf("(%s) offset = %d, compressed data size = %d, flags = %x, reader version = %d", entry.Name, offset, entry.CompressedSize64, entry.Flags, entry.ReaderVersion)
			}
		}

		err = cb(index, entry.Name, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.Wrap(err, 0)
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
		return errors.Wrap(err, 0)
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
			return errors.Wrap(err, 0)
		}

		err = cb(index, entry.Name, int64(entry.UncompressedSize64), rc, numEntries)
		rc.Close()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

// utils

func printExtras(consumer *state.Consumer, size int64, compressedSize int64, uncompressedSize int64, comment string) {
	consumer.Infof("Comment: (%s)", comment)
	consumer.Infof("Sizes: ")
	consumer.Infof(" → Archive size      : %s (%d bytes)", humanize.IBytes(uint64(size)), size)
	consumer.Infof(" → Sum (compressed)  : %s (%d bytes)", humanize.IBytes(uint64(compressedSize)), compressedSize)
	consumer.Infof(" → Sum (uncompressed): %s (%d bytes)", humanize.IBytes(uint64(uncompressedSize)), uncompressedSize)
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
