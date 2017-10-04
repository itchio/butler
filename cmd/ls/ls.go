package ls

import (
	"encoding/binary"
	"io"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/arkive/zip"
	"github.com/itchio/butler/butler"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
)

var args = struct {
	file *string
}{}

func Register(ctx *butler.Context) {
	cmd := ctx.App.Command("ls", "Prints the list of files, dirs and symlinks contained in a patch file, signature file, or archive")
	args.file = cmd.Arg("file", "A file you'd like to list the contents of").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *butler.Context) {
	ctx.Must(Do(ctx, *args.file))
}

func Do(ctx *butler.Context, inPath string) error {
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

	if stats.IsDir() {
		comm.Logf("%s: directory", path)
		return nil
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return nil
	}

	log := func(line string) {
		comm.Logf(line)
	}

	var magic int32
	err = binary.Read(reader, wire.Endianness, &magic)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	switch magic {
	case pwr.PatchMagic:
		{
			h := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(reader)
			err = rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			log("pre-patch container:")
			container.Print(log)

			container.Reset()
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			log("================================")
			log("post-patch container:")
			container.Print(log)
		}

	case pwr.SignatureMagic:
		{
			h := &pwr.SignatureHeader{}
			rctx := wire.NewReadContext(reader)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)
		}

	case pwr.ManifestMagic:
		{
			h := &pwr.ManifestHeader{}
			rctx := wire.NewReadContext(reader)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.Wrap(err, 0)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)
		}

	case pwr.WoundsMagic:
		{
			wh := &pwr.WoundsHeader{}
			rctx := wire.NewReadContext(reader)
			err := rctx.ReadMessage(wh)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			container.Print(log)

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
				comm.Logf(wound.PrettyString(container))
			}
		}

	default:
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		wasZip := func() bool {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					ctx.Must(err)
				}
				return false
			}

			container, err := tlc.WalkZip(zr, func(fi os.FileInfo) bool { return true })
			ctx.Must(err)
			container.Print(log)
			return true
		}()

		if !wasZip {
			comm.Logf("%s: not sure - try the file(1) command if your system has it!", path)
		}
	}

	return nil
}
