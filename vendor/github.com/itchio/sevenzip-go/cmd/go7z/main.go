package main

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/httpkit/progress"
	"github.com/itchio/sevenzip-go/sz"
	"github.com/pkg/errors"
)

type ecs struct {
	// muffin
}

func main() {
	lib, err := sz.NewLib()
	must(err)
	log.Printf("Initialized 7-zip %s...", lib.GetVersion())
	defer lib.Free()

	args := os.Args[1:]

	if len(args) < 1 {
		log.Printf("Usage: go7z ARCHIVE")
		os.Exit(1)
	}

	inPath := args[0]
	ext := filepath.Ext(inPath)
	if ext != "" {
		ext = ext[1:]
	}
	log.Printf("ext = %s", ext)

	f, err := os.Open(inPath)
	must(err)

	stats, err := f.Stat()
	must(err)

	is, err := sz.NewInStream(f, ext, stats.Size())
	must(err)
	log.Printf("Created input stream (%s, %d bytes)...", inPath, stats.Size())

	is.Stats = &sz.ReadStats{}

	a, err := lib.OpenArchive(is, false)
	if err != nil {
		log.Printf("Could not open archive by ext, trying by signature")

		_, err = is.Seek(0, io.SeekStart)
		must(err)

		a, err = lib.OpenArchive(is, true)
	}
	must(err)

	log.Printf("Opened archive: format is (%s)", a.GetArchiveFormat())

	itemCount, err := a.GetItemCount()
	must(err)
	log.Printf("Archive has %d items", itemCount)

	ec, err := sz.NewExtractCallback(&ecs{})
	must(err)
	defer ec.Free()

	var indices = make([]int64, itemCount)
	for i := 0; i < int(itemCount); i++ {
		indices[i] = int64(i)
	}
	middle := itemCount / 2

	log.Printf("Doing first half...")
	err = a.ExtractSeveral(indices[0:middle], ec)
	must(err)

	for i := 0; i < 15; i++ {
		is.Stats.RecordRead(0, 0)
	}

	log.Printf("Doing second half...")
	err = a.ExtractSeveral(indices[middle:], ec)
	must(err)

	errs := ec.Errors()
	if len(errs) > 0 {
		log.Printf("There were %d errors during extraction:", len(errs))
		for _, err := range errs {
			log.Printf("- %s", err.Error())
		}
	}

	width := len(is.Stats.Reads)
	height := 800
	log.Printf("Making %dx%d image", width, height)

	rect := image.Rect(0, 0, width, height)
	img := image.NewRGBA(rect)

	black := &color.RGBA{
		R: 0,
		G: 0,
		B: 0,
		A: 255,
	}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, black)
		}
	}

	scale := 1.0 / float64(stats.Size()) * float64(height)
	c := &color.RGBA{
		R: 255,
		G: 0,
		B: 0,
		A: 255,
	}

	var maxReadSize int64 = 1
	for _, op := range is.Stats.Reads {
		if op.Size > maxReadSize {
			maxReadSize = op.Size
		}
	}

	for x, op := range is.Stats.Reads {
		ymin := int(math.Floor(float64(op.Offset) * scale))
		ymax := int(math.Ceil(float64(op.Offset+op.Size) * scale))

		cd := *c
		cd.G = uint8(float64(op.Size) / float64(maxReadSize) * 255)

		for y := ymin; y <= ymax; y++ {
			img.Set(x, y, &cd)
		}
	}

	imageFile, err := os.Create("out/reads.png")
	must(err)
	defer imageFile.Close()

	err = png.Encode(imageFile, img)
	must(err)
}

func must(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func (e *ecs) GetStream(item *sz.Item) (*sz.OutStream, error) {
	propPath, ok := item.GetStringProperty(sz.PidPath)
	if !ok {
		return nil, errors.New("could not get item path")
	}

	outPath := filepath.ToSlash(propPath)
	// Remove illegal character for windows paths, see
	// https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx
	for i := byte(0); i <= 31; i++ {
		outPath = strings.Replace(outPath, string([]byte{i}), "_", -1)
	}

	absoluteOutPath := filepath.Join("out", outPath)

	log.Printf("  ")
	log.Printf("==> Extracting %d: %s", item.GetArchiveIndex(), outPath)

	if attrib, ok := item.GetUInt64Property(sz.PidAttrib); ok {
		log.Printf("==> Attrib       %08x", attrib)
	}
	if attrib, ok := item.GetUInt64Property(sz.PidPosixAttrib); ok {
		log.Printf("==> Posix Attrib %08x", attrib)
	}
	if symlink, ok := item.GetStringProperty(sz.PidSymLink); ok {
		log.Printf("==> Symlink dest: %s", symlink)
	}

	isDir, _ := item.GetBoolProperty(sz.PidIsDir)
	if isDir {
		log.Printf("Making %s", outPath)

		err := os.MkdirAll(absoluteOutPath, 0755)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		// is a dir, just skip it
		return nil, nil
	}

	err := os.MkdirAll(filepath.Dir(absoluteOutPath), 0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	of, err := os.Create(absoluteOutPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	os, err := sz.NewOutStream(of)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return os, nil
}

func (e *ecs) SetProgress(complete int64, total int64) {
	log.Printf("Progress: %s / %s",
		progress.FormatBytes(complete),
		progress.FormatBytes(total),
	)
}
