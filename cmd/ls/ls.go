package ls

import (
	"archive/tar"
	"encoding/binary"
	"io"
	"os"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/arkive/zip"
	"github.com/itchio/boar"

	"github.com/itchio/headway/united"

	"github.com/itchio/savior"
	"github.com/itchio/savior/seeksource"

	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/wire"

	"github.com/itchio/lake/tlc"

	"github.com/pkg/errors"
)

var args = struct {
	file *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("ls", "Prints the list of files, dirs and symlinks contained in a patch file, signature file, or archive")
	args.file = cmd.Arg("file", "A file you'd like to list the contents of").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.file))
}

func Do(ctx *mansion.Context, inPath string) error {
	consumer := comm.NewStateConsumer()

	reader, err := eos.Open(inPath, option.WithConsumer(consumer))
	if err != nil {
		return errors.WithStack(err)
	}

	path := eos.Redact(inPath)

	defer reader.Close()

	stats, err := reader.Stat()
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory", path)
	}
	if err != nil {
		return errors.WithStack(err)
	}

	log := func(line string) {
		comm.Logf(line)
	}

	if stats.IsDir() {
		walkOpts := &tlc.WalkOpts{
			Filter: filtering.FilterPaths,
		}
		walkOpts.AutoWrap(&inPath, consumer)

		container, err := tlc.WalkDir(inPath, walkOpts)
		if err != nil {
			return errors.WithStack(err)
		}

		comm.Logf("%s: directory", path)
		container.Print(log)
		return nil
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return nil
	}

	source := seeksource.FromFile(reader)

	_, err = source.Resume(nil)
	if err != nil {
		return errors.WithStack(err)
	}

	var magic int32
	err = binary.Read(source, wire.Endianness, &magic)
	if err != nil {
		return errors.Wrap(err, "reading magic number")
	}

	switch magic {
	case pwr.PatchMagic:
		{
			h := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(source)
			err = rctx.ReadMessage(h)
			if err != nil {
				return errors.WithStack(err)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.WithStack(err)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}

			log("pre-patch container:")
			container.Print(log)

			container.Reset()
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}

			log("================================")
			log("post-patch container:")
			container.Print(log)
		}

	case pwr.SignatureMagic:
		{
			h := &pwr.SignatureHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.WithStack(err)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.WithStack(err)
			}
			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}
			container.Print(log)
		}

	case pwr.ManifestMagic:
		{
			h := &pwr.ManifestHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(h)
			if err != nil {
				return errors.WithStack(err)
			}

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			if err != nil {
				return errors.WithStack(err)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}
			container.Print(log)
		}

	case pwr.WoundsMagic:
		{
			wh := &pwr.WoundsHeader{}
			rctx := wire.NewReadContext(source)
			err := rctx.ReadMessage(wh)
			if err != nil {
				return errors.WithStack(err)
			}

			container := &tlc.Container{}
			err = rctx.ReadMessage(container)
			if err != nil {
				return errors.WithStack(err)
			}
			container.Print(log)

			for {
				wound := &pwr.Wound{}
				err = rctx.ReadMessage(wound)
				if err != nil {
					if errors.Cause(err) == io.EOF {
						break
					} else {
						return errors.WithStack(err)
					}
				}
				comm.Logf(wound.PrettyString(container))
			}
		}

	default:
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.WithStack(err)
		}

		wasZip := func() bool {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					ctx.Must(err)
				}
				return false
			}

			container, err := tlc.WalkZip(zr, &tlc.WalkOpts{
				Filter: func(fi os.FileInfo) bool { return true },
			})
			ctx.Must(err)
			container.Print(log)

			err = container.Validate()
			if err != nil {
				comm.Notice("Validation failed", []string{"One or more errors found, see below"})
			}
			comm.Logf("%s", err)

			return true
		}()

		if wasZip {
			return nil
		}

		_, err = reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.WithStack(err)
		}

		wasTar := func() bool {
			tr := tar.NewReader(reader)

			for {
				hdr, err := tr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}
					return false
				}

				comm.Logf("%s %10s %s", os.FileMode(hdr.Mode), united.FormatBytes(hdr.Size), hdr.Name)
			}
			return true
		}()

		if wasTar {
			return nil
		}

		_, err = reader.Seek(0, io.SeekStart)
		if err != nil {
			return errors.WithStack(err)
		}

		wasBoar := func() bool {
			numEntries := 0
			info, err := boar.Probe(&boar.ProbeParams{
				File:     reader,
				Consumer: consumer,
				OnEntries: func(entries []*savior.Entry) {
					numEntries += len(entries)
					for _, e := range entries {
						comm.Logf("%s %10s %s", e.Mode, united.FormatBytes(e.UncompressedSize), e.CanonicalPath)
					}
				},
			})
			if err != nil {
				consumer.Warnf("Couldn't probe with boar: %+v", err)
				return false
			}

			if info.Strategy == boar.StrategyDmg {
				consumer.Errorf("Listing dmg files is deprecated, sorry!")
				return false
			}

			if numEntries == 0 {
				consumer.Warnf("Opened with boar successfully, but had 0 entries.")
				consumer.Warnf("Archive info was: %s", info)
			}

			return true
		}()

		if wasBoar {
			return nil
		}

		comm.Logf("%s: not able to list contents", path)
	}

	return nil
}
