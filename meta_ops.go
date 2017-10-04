package main

import (
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/fasterthanlime/spellbook"
	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
	"github.com/itchio/arkive/zip"

	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/kardianos/osext"
)

// A ContainerResult is sent in json mode by the file command
type ContainerResult struct {
	Type             string   `json:"type"`
	Spell            []string `json:"spell"`
	NumFiles         int      `json:"numFiles"`
	NumDirs          int      `json:"numDirs"`
	NumSymlinks      int      `json:"numSymlinks"`
	UncompressedSize int64    `json:"uncompressedSize"`
}

func which() {
	p, err := osext.Executable()
	must(err)

	comm.Logf("You're running butler %s, from the following path:", versionString)
	comm.Logf("%s", p)
}

func file(inPath string) {
	reader, err := eos.Open(inPath)
	must(err)

	path := eos.Redact(inPath)

	defer reader.Close()

	stats, err := reader.Stat()
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory", path)
	}
	must(err)

	result := ContainerResult{
		Type: "unknown",
	}

	sr := wizutil.NewSliceReader(reader, 0, stats.Size())
	spell := spellbook.Identify(sr, 0)
	if spell != nil {
		result.Type = "other"
		result.Spell = spell
		comm.Logf("%s: %s", path, wizutil.MergeStrings(spell))
	}

	if stats.IsDir() {
		comm.Logf("%s: directory", path)
		return
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return
	}

	prettySize := humanize.IBytes(uint64(stats.Size()))

	_, err = reader.Seek(0, io.SeekStart)
	must(err)

	var magic int32
	must(binary.Read(reader, wire.Endianness, &magic))

	switch magic {
	case pwr.PatchMagic:
		{
			ph := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(ph))

			rctx, err = pwr.DecompressWire(rctx, ph.GetCompression())
			must(err)
			container := &tlc.Container{}
			must(rctx.ReadMessage(container)) // target container
			container.Reset()
			must(rctx.ReadMessage(container)) // source container

			comm.Logf("%s: %s wharf patch file (%s) with %s", path, prettySize, ph.GetCompression().ToString(), container.Stats())
			result = ContainerResult{
				Type:             "wharf/patch",
				NumFiles:         len(container.Files),
				NumDirs:          len(container.Dirs),
				NumSymlinks:      len(container.Symlinks),
				UncompressedSize: container.Size,
			}
		}

	case pwr.SignatureMagic:
		{
			sh := &pwr.SignatureHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(sh))

			rctx, err = pwr.DecompressWire(rctx, sh.GetCompression())
			must(err)
			container := &tlc.Container{}
			must(rctx.ReadMessage(container))

			comm.Logf("%s: %s wharf signature file (%s) with %s", path, prettySize, sh.GetCompression().ToString(), container.Stats())
			result = ContainerResult{
				Type:             "wharf/signature",
				NumFiles:         len(container.Files),
				NumDirs:          len(container.Dirs),
				NumSymlinks:      len(container.Symlinks),
				UncompressedSize: container.Size,
			}
		}

	case pwr.ManifestMagic:
		{
			mh := &pwr.ManifestHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(mh))

			rctx, err = pwr.DecompressWire(rctx, mh.GetCompression())
			must(err)
			container := &tlc.Container{}
			must(rctx.ReadMessage(container))

			comm.Logf("%s: %s wharf manifest file (%s) with %s", path, prettySize, mh.GetCompression().ToString(), container.Stats())
			result = ContainerResult{
				Type:             "wharf/manifest",
				NumFiles:         len(container.Files),
				NumDirs:          len(container.Dirs),
				NumSymlinks:      len(container.Symlinks),
				UncompressedSize: container.Size,
			}
		}

	case pwr.WoundsMagic:
		{
			wh := &pwr.WoundsHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(wh))

			container := &tlc.Container{}
			must(rctx.ReadMessage(container))

			files := make(map[int64]bool)
			totalWounds := int64(0)

			for {
				wound := &pwr.Wound{}

				err = rctx.ReadMessage(wound)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					} else {
						must(err)
					}
				}

				if wound.Kind == pwr.WoundKind_FILE {
					totalWounds += (wound.End - wound.Start)
					files[wound.Index] = true
				}
			}

			comm.Logf("%s: %s wharf wounds file with %s, %s wounds in %d files", path, prettySize, container.Stats(),
				humanize.IBytes(uint64(totalWounds)), len(files))
			result = ContainerResult{
				Type: "wharf/wounds",
			}
		}

	default:
		_, err := reader.Seek(0, io.SeekStart)
		must(err)

		func() {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					must(err)
				}
				return
			}

			container, err := tlc.WalkZip(zr, func(fi os.FileInfo) bool { return true })
			must(err)

			prettyUncompressed := humanize.IBytes(uint64(container.Size))
			comm.Logf("%s: %s zip file with %s, %s uncompressed", path, prettySize, container.Stats(), prettyUncompressed)
			result = ContainerResult{
				Type:             "zip",
				NumFiles:         len(container.Files),
				NumDirs:          len(container.Dirs),
				NumSymlinks:      len(container.Symlinks),
				UncompressedSize: container.Size,
			}
		}()

		if result.Type == "unknown" {
			comm.Logf("%s: not sure - try the file(1) command if your system has it!", path)
		}
		comm.Result(result)
	}
}

// WalkResult is sent for each item that's walked
type WalkResult struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Size int64  `json:"size,omitempty"`
}

func walk(dir string) {
	startTime := time.Now()

	container, err := tlc.WalkDir(dir, func(fi os.FileInfo) bool { return true })
	must(err)

	totalEntries := 0
	send := func(path string) {
		totalEntries++
		comm.Result(&WalkResult{
			Type: "entry",
			Path: path,
		})

		comm.Debugf("%s", path)
	}

	for _, f := range container.Files {
		send(f.Path)
	}

	for _, s := range container.Symlinks {
		send(s.Path)
	}

	comm.Result(&WalkResult{
		Type: "totalSize",
		Size: container.Size,
	})

	comm.Statf("%d entries (%s) walked in %s", totalEntries, humanize.IBytes(uint64(container.Size)), time.Since(startTime))
}
