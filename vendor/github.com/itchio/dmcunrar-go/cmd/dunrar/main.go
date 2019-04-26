package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/itchio/go-itchio/itchfs"
	"github.com/itchio/wharf/counter"
	"github.com/itchio/wharf/eos"

	"github.com/itchio/dmcunrar-go/dmcunrar"
	"github.com/itchio/httpkit/progress"
)

func main() {
	exec := os.Args[0]
	args := os.Args[1:]
	verbose := false

	var nameArgs []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-v":
				verbose = true
			default:
				log.Fatalf("unsupported flag: %s", arg)
			}
		} else {
			nameArgs = append(nameArgs, arg)
		}
	}

	if len(nameArgs) < 1 {
		log.Fatalf("usage: %s FILE.rar", exec)
	}
	name := nameArgs[0]

	eos.RegisterHandler(&itchfs.ItchFS{})

	log.Printf("Opening file...")
	f, err := eos.Open(name)
	must(err)
	defer f.Close()

	stats, err := f.Stat()
	must(err)

	log.Printf("Opening as RAR archive...")
	archive, err := dmcunrar.OpenArchive(f, stats.Size())
	must(err)
	defer archive.Free()

	log.Printf("Listing contents...")
	numUnsupported := 0

	var uncompressedSize int64
	for i := int64(0); i < archive.GetFileCount(); i++ {
		stat := archive.GetFileStat(i)
		if stat != nil {
			uncompressedSize += stat.GetUncompressedSize()
		}

		if !archive.FileIsDirectory(i) {
			err := archive.FileIsSupported(i)
			if err != nil {
				numUnsupported++
				name, _ := archive.GetFilename(i)
				log.Printf("%+v", errors.WithMessagef(err, "for file %s", name))
			}
		}
	}
	log.Printf("Extracting %d files, %s uncompressed", archive.GetFileCount(), progress.FormatBytes(uncompressedSize))

	extractEntry := func(i int64) {
		name, _ := archive.GetFilename(i)

		stat := archive.GetFileStat(i)
		if stat == nil {
			must(errors.New("null file stat!"))
		}

		if verbose {
			log.Printf("Extracting (%s)", name)
			log.Printf("Stat: %#v", stat)
		}

		dest := filepath.Join("out", name)
		if archive.FileIsDirectory(i) {
			os.MkdirAll(dest, 0755)
			return
		}

		must(os.MkdirAll(filepath.Dir(dest), 0755))

		f, err := os.Create(dest)
		must(err)
		defer f.Close()

		size := stat.GetUncompressedSize()

		tracker := progress.NewTracker()
		tracker.Bar().ShowCounters = false
		label := dest
		maxLabel := 40
		if len(label) > maxLabel {
			label = label[len(label)-maxLabel:]
		}
		tracker.Bar().Postfix(label)
		tracker.SetTotalBytes(size)
		tracker.Start()
		defer tracker.Finish()

		cw := counter.NewWriterCallback(func(count int64) {
			tracker.SetProgress(float64(count) / float64(size))
		}, f)

		ef := dmcunrar.NewExtractedFile(cw)
		defer ef.Free()

		must(archive.ExtractFile(ef, i))
	}

	for i := int64(0); i < archive.GetFileCount(); i++ {
		extractEntry(i)
	}

	if numUnsupported > 0 {
		must(errors.Errorf("had %d unsupported files", numUnsupported))
	}
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
}
