package main

import (
	"archive/zip"
	"encoding/binary"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/kardianos/osext"
)

func which() {
	p, err := osext.Executable()
	must(err)

	comm.Logf("You're running butler %s, from the following path:", versionString)
	comm.Logf("%s", p)
}

func file(path string) {
	stats, err := os.Lstat(path)
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory")
	}
	must(err)

	if stats.IsDir() {
		comm.Logf("%s: directory", path)
		return
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return
	}

	prettySize := humanize.Bytes(uint64(stats.Size()))

	reader, err := os.Open(path)
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
		}

	default:
		_, err := reader.Seek(0, os.SEEK_SET)
		must(err)

		wasZip := func() bool {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					must(err)
				}
				return false
			}

			container, err := tlc.WalkZip(zr, func(fi os.FileInfo) bool { return true })
			must(err)

			comm.Logf("%s: %s zip file with %s", path, prettySize, container.Stats())
			return true
		}()

		if !wasZip {
			comm.Logf("%s: not sure - try the file(1) command if your system has it!", path)
		}
	}
}

func ls(path string) {
	stats, err := os.Lstat(path)
	if os.IsNotExist(err) {
		comm.Dief("%s: no such file or directory")
	}
	must(err)

	if stats.IsDir() {
		comm.Logf("%s: directory", path)
		return
	}

	if stats.Size() == 0 {
		comm.Logf("%s: empty file. peaceful.", path)
		return
	}

	log := func(line string) {
		comm.Logf(line)
	}

	reader, err := os.Open(path)
	must(err)

	var magic int32
	must(binary.Read(reader, wire.Endianness, &magic))

	switch magic {
	case pwr.PatchMagic:
		{
			h := &pwr.PatchHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(h))

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			must(err)
			container := &tlc.Container{}
			must(rctx.ReadMessage(container))
			log("pre-patch container:")
			container.Print(log)

			container.Reset()
			must(rctx.ReadMessage(container))
			log("================================")
			log("post-patch container:")
			container.Print(log)
		}

	case pwr.SignatureMagic:
		{
			h := &pwr.SignatureHeader{}
			rctx := wire.NewReadContext(reader)
			must(rctx.ReadMessage(h))

			rctx, err = pwr.DecompressWire(rctx, h.GetCompression())
			must(err)
			container := &tlc.Container{}
			must(rctx.ReadMessage(container))
			container.Print(log)
		}

	default:
		_, err := reader.Seek(0, os.SEEK_SET)
		must(err)

		wasZip := func() bool {
			zr, err := zip.NewReader(reader, stats.Size())
			if err != nil {
				if err != zip.ErrFormat {
					must(err)
				}
				return false
			}

			container, err := tlc.WalkZip(zr, func(fi os.FileInfo) bool { return true })
			must(err)
			container.Print(log)
			return true
		}()

		if !wasZip {
			comm.Logf("%s: not sure - try the file(1) command if your system has it!", path)
		}
	}
}
