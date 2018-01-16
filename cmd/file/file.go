package file

import (
	"encoding/binary"
	"io"
	"os"

	humanize "github.com/dustin/go-humanize"
	"github.com/fasterthanlime/spellbook"
	"github.com/fasterthanlime/wizardry/wizardry/wizutil"
	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("file", "Prints the type of a given file, and some stats about it")
	args.file = cmd.Arg("file", "A file you'd like to identify").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.file))
}

func Do(ctx *mansion.Context, inPath string) error {
	reader, err := eos.Open(inPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	path := eos.Redact(inPath)

	defer reader.Close()

	stats, err := reader.Stat()
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory", path)
	}
	if err != nil {
		return errors.Wrap(err, 0)
	}

	result := mansion.ContainerResult{
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
		return nil
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return nil
	}

	prettySize := humanize.IBytes(uint64(stats.Size()))

	source := seeksource.FromFile(reader)

	_, err = source.Resume(nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var magic int32
	err = binary.Read(source, wire.Endianness, &magic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch magic {
	case pwr.PatchMagic:
		{
			ph := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(ph)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, ph.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container) // target container
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Reset()
			err = rctx.ReadMessage(container) // source container
			if err != nil {
				return errors.Wrap(err, 0)
			}

			comm.Logf("%s: %s wharf patch file (%s) with %s", path, prettySize, ph.GetCompression().ToString(), container.Stats())
			result = mansion.ContainerResult{
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
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(sh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, sh.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			comm.Logf("%s: %s wharf signature file (%s) with %s", path, prettySize, sh.GetCompression().ToString(), container.Stats())
			result = mansion.ContainerResult{
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
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(mh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, mh.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			comm.Logf("%s: %s wharf manifest file (%s) with %s", path, prettySize, mh.GetCompression().ToString(), container.Stats())
			result = mansion.ContainerResult{
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
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(wh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			files := make(map[int64]bool)
			totalWounds := int64(0)

			for {
				wound := &pwr.Wound{}

				err = rctx.ReadMessage(wound)
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					} else {
						return errors.Wrap(err, 0)
					}
				}

				if wound.Kind == pwr.WoundKind_FILE {
					totalWounds += (wound.End - wound.Start)
					files[wound.Index] = true
				}
			}

			comm.Logf("%s: %s wharf wounds file with %s, %s wounds in %d files", path, prettySize, container.Stats(),
				humanize.IBytes(uint64(totalWounds)), len(files))
			result = mansion.ContainerResult{
				Type: "wharf/wounds",
			}
		}

	default:
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		func() {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					ctx.Must(err)
				}
				return
			}

			container, err := tlc.WalkZip(zr, &tlc.WalkOpts{
				Filter: func(fi os.FileInfo) bool { return true },
			})
			ctx.Must(err)

			prettyUncompressed := humanize.IBytes(uint64(container.Size))
			comm.Logf("%s: %s zip file with %s, %s uncompressed", path, prettySize, container.Stats(), prettyUncompressed)
			result = mansion.ContainerResult{
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

	return nil
}
