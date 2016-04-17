package main

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/blang/semver"
	"github.com/dustin/go-humanize"
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

func queryLatestVersion() (*semver.Version, *semver.Version, error) {
	if *appArgs.quiet {
		return nil, nil, nil
	}

	if version == "head" {
		comm.Opf("Bleeding-edge, skipping version check")
		return nil, nil, nil
	}

	currentVer, err := semver.Make(version)
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

	latestVersion := strings.TrimLeft(strings.Trim(string(buf), " \r\n"), "v")

	latestVer, err := semver.Make(latestVersion)
	if err != nil {
		return nil, nil, err
	}

	return &currentVer, &latestVer, nil
}

func upgrade(assumeYes bool) {
	must(doUpgrade(assumeYes))
}

func doUpgrade(assumeYes bool) error {
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

	if !assumeYes {
		fmt.Printf("\n:: Do you want to upgrade now? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.ToLower(scanner.Text())

		if answer != "y" {
			fmt.Println("Okay, not upgrading. Bye!")
			return nil
		}
	}

	execPath, err := osext.Executable()
	must(err)

	oldPath := execPath + ".old"
	newPath := execPath + ".new"

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	execURL := fmt.Sprintf("%s/v%s/butler%s", updateBaseURL, latestVer.String(), ext)
	comm.Opf("%s", execURL)

	dl(execURL, newPath)
	must(os.Chmod(newPath, os.FileMode(0755)))

	comm.Opf("Backing up current version to %s just in case...", oldPath)
	must(os.Rename(execPath, oldPath))

	must(os.Rename(newPath, execPath))
	must(os.Remove(oldPath))

	comm.Statf("Upgraded butler from %s to %s. Have a nice day!", version, latestVer)
	return nil
}
