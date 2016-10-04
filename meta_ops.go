package main

import (
	"archive/zip"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/go-itchio"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wire"
	"github.com/kardianos/osext"
)

var updateBaseURL = fmt.Sprintf("https://dl.itch.ovh/butler/%s-%s", runtime.GOOS, runtime.GOARCH)

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

	prettySize := humanize.IBytes(uint64(stats.Size()))

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
				totalWounds += (wound.End - wound.Start)
				files[wound.FileIndex] = true
			}

			comm.Logf("%s: %s wharf wounds file with %s, %s wounds in %d files", path, prettySize, container.Stats(),
				humanize.IBytes(uint64(totalWounds)), len(files))
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

	case pwr.ManifestMagic:
		{
			h := &pwr.ManifestHeader{}
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

func versionCheck() {
	currentVer, latestVer, err := queryLatestVersion()
	if err != nil {
		comm.Logf("Version check failed: %s", err.Error())
	}

	if currentVer == nil || latestVer == nil {
		return
	}

	if latestVer.GT(*currentVer) {
		comm.Notice("New version available",
			[]string{
				fmt.Sprintf("Current version: %s", version),
				fmt.Sprintf("Latest version:  %s", latestVer),
				"",
				"Run `butler upgrade` to get it.",
			})
	}
}

func parseSemver(s string) (semver.Version, error) {
	return semver.Make(strings.TrimLeft(s, "v"))
}

func queryLatestVersion() (*semver.Version, *semver.Version, error) {
	if *appArgs.quiet {
		return nil, nil, nil
	}

	if version == "head" {
		return nil, nil, nil
	}

	currentVer, err := parseSemver(version)
	if err != nil {
		return nil, nil, err
	}

	c := itchio.ClientWithKey("x")

	latestURL := fmt.Sprintf("%s/LATEST", updateBaseURL)
	req, err := http.NewRequest("GET", latestURL, nil)
	if err != nil {
		return nil, nil, err
	}

	res, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode != 200 {
		err := fmt.Errorf("HTTP %d: %s", res.StatusCode, latestURL)
		return nil, nil, err
	}

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	latestVersion := strings.Trim(string(buf), " \r\n")
	latestVer, err := parseSemver(latestVersion)
	if err != nil {
		return nil, nil, err
	}

	return &currentVer, &latestVer, nil
}

func upgrade(head bool) {
	must(doUpgrade(head))
}

func doUpgrade(head bool) error {
	if head {
		if !comm.YesNo("Do you want to upgrade to the bleeding-edge version? Things may break!") {
			comm.Logf("Okay, not upgrading. Bye!")
			return nil
		}

		return applyUpgrade("head", "head")
	}

	if version == "head" {
		comm.Statf("Bleeding-edge, not upgrading unless told to.")
		comm.Logf("(Use `--head` if you want to upgrade to the latest bleeding-edge version)")
		return nil
	}

	comm.Opf("Looking for upgrades...")

	currentVer, latestVer, err := queryLatestVersion()
	if err != nil {
		return fmt.Errorf("Version check failed: %s", err.Error())
	}

	if latestVer == nil || currentVer.GTE(*latestVer) {
		comm.Statf("Your butler is up-to-date. Have a nice day!")
		return nil
	}

	comm.Statf("Current version: %s", currentVer.String())
	comm.Statf("Latest version : %s", latestVer.String())

	if !comm.YesNo("Do you want to upgrade now?") {
		comm.Logf("Okay, not upgrading. Bye!")
		return nil
	}

	must(applyUpgrade(currentVer.String(), latestVer.String()))
	return nil
}

func applyUpgrade(before string, after string) error {
	execPath, err := osext.Executable()
	if err != nil {
		return err
	}

	oldPath := execPath + ".old"
	newPath := execPath + ".new"
	gzPath := newPath + ".gz"

	err = os.RemoveAll(newPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(gzPath)
	if err != nil {
		return err
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	fragment := fmt.Sprintf("v%s", after)
	if after == "head" {
		fragment = "head"
	}
	execURL := fmt.Sprintf("%s/%s/butler%s", updateBaseURL, fragment, ext)

	gzURL := fmt.Sprintf("%s/%s/butler.gz", updateBaseURL, fragment)
	comm.Opf("%s", gzURL)

	err = func() error {
		_, err := tryDl(gzURL, gzPath)
		if err != nil {
			return err
		}

		fr, err := os.Open(gzPath)
		if err != nil {
			return err
		}
		defer fr.Close()

		gr, err := gzip.NewReader(fr)
		if err != nil {
			return err
		}

		fw, err := os.Create(newPath)
		if err != nil {
			return err
		}
		defer fw.Close()

		_, err = io.Copy(fw, gr)
		if err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		comm.Opf("Falling back to %s", execURL)
		_, err = tryDl(execURL, newPath)
		must(err)
	}

	err = os.Chmod(newPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	comm.Opf("Backing up current version to %s just in case...", oldPath)
	err = os.Rename(execPath, oldPath)
	if err != nil {
		return err
	}

	err = os.Rename(newPath, execPath)
	if err != nil {
		return err
	}

	err = os.Remove(oldPath)
	if err != nil {
		if os.IsPermission(err) && runtime.GOOS == "windows" {
			// poor windows doesn't like us removing executables from under it
			// I vote we move on and let butler.exe.old hang around.
		} else {
			return err
		}
	}

	comm.Statf("Upgraded butler from %s to %s. Have a nice day!", before, after)
	return nil
}
