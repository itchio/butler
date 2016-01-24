package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

// #cgo windows LDFLAGS: -Wl,--allow-multiple-definition -static
import "C"

var (
	version = "head" // set by command-line on CI release builds
	app     = kingpin.New("butler", "Your very own itch.io helper")

	dlCmd        = app.Command("dl", "Download a file (resumes if can, checks hashes)")
	pushCmd      = app.Command("push", "Upload a new version of something to itch.io")
	untarCmd     = app.Command("untar", "Extract a .tar file")
	wipeCmd      = app.Command("wipe", "Completely remove a directory (rm -rf)")
	dittoCmd     = app.Command("ditto", "Create a mirror (incl. symlinks) of a directory into another dir (rsync -az)")
	mkdirCmd     = app.Command("mkdir", "Create an empty directory and all required parent directories (mkdir -p)")
	megatestCmd  = app.Command("megatest", "Test megafile").Hidden()
	megadiffCmd  = app.Command("megadiff", "Generate a patch to turn 'target' into 'source'").Hidden()
	megapatchCmd = app.Command("megapatch", "Applies 'patch' to 'target' and writes it to 'output'").Hidden()
)

var appArgs = struct {
	json       *bool
	quiet      *bool
	verbose    *bool
	timestamps *bool
	csv        *bool
	paranoid   *bool
}{
	app.Flag("json", "Enable machine-readable JSON-lines output").Short('j').Bool(),
	app.Flag("quiet", "Hide progress indicators & other extra info").Short('q').Bool(),
	app.Flag("verbose", "Display as much extra info as possible").Short('v').Bool(),
	app.Flag("timestamps", "Prefix all output by timestamps (for logging purposes)").Bool(),
	app.Flag("csv", "Output stats in CSV format").Bool(),
	app.Flag("paranoid", "Insist on checking all available hashes, not just the fastest ones").Bool(),
}

var dlArgs = struct {
	url  *string
	dest *string
}{
	dlCmd.Arg("url", "Address to download from").Required().String(),
	dlCmd.Arg("dest", "File to write downloaded data to").Required().String(),
}

var pushArgs = struct {
	src      *string
	repo     *string
	identity *string
	address  *string
}{
	pushCmd.Arg("src", "Directory or zip archive to upload, e.g.").Required().ExistingFileOrDir(),
	pushCmd.Arg("repo", "Repository to push to, e.g. leafo/xmoon:win64").Required().String(),
	pushCmd.Flag("identity", "Path to the private key used for public key authentication.").Default(fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".ssh/id_rsa")).Short('i').ExistingFile(),
	pushCmd.Flag("address", "Specify wharf address (advanced)").Default("wharf.itch.zone").Short('a').Hidden().String(),
}

var untarArgs = struct {
	file *string
	dir  *string
}{
	untarCmd.Arg("file", "Path of the .tar archive to extract").Required().String(),
	untarCmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String(),
}

var wipeArgs = struct {
	path *string
}{
	wipeCmd.Arg("path", "Path to completely remove, including its contents").Required().String(),
}

var mkdirArgs = struct {
	path *string
}{
	mkdirCmd.Arg("path", "Directory to create").Required().String(),
}

var dittoArgs = struct {
	src *string
	dst *string
}{
	dittoCmd.Arg("src", "Directory to mirror").Required().String(),
	dittoCmd.Arg("dst", "Path where to create a mirror").Required().String(),
}

var megatestArgs = struct {
	src *string
}{
	megatestCmd.Arg("src", "Directory to walk").Required().String(),
}

var megadiffArgs = struct {
	target *string
	source *string
	patch  *string
	verify *bool
}{
	megadiffCmd.Arg("target", "Directory with older files").Required().String(),
	megadiffCmd.Arg("source", "Directory with newer files").Required().String(),
	megadiffCmd.Arg("patch", "Where to write the patch file").Default("patch.dat").String(),
	megadiffCmd.Flag("verify", "Verify that patch applies cleanly").Bool(),
}

var megapatchArgs = struct {
	patch  *string
	target *string
	output *string
}{
	megapatchCmd.Arg("patch", "Patch file").Required().String(),
	megapatchCmd.Arg("target", "Directory with older files").Required().String(),
	megapatchCmd.Arg("source", "Path to create directory with newer files").Required().String(),
}

func must(err error) {
	if err != nil {
		Die(err.Error())
	}
}

func main() {
	app.HelpFlag.Short('h')
	app.Version(version)
	app.VersionFlag.Short('V')

	cmd, err := app.Parse(os.Args[1:])
	if *appArgs.timestamps {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}

	if *appArgs.csv {
		defer CsvFinish()
	}

	switch kingpin.MustParse(cmd, err) {
	case dlCmd.FullCommand():
		dl(*dlArgs.url, *dlArgs.dest)

	case pushCmd.FullCommand():
		push(*pushArgs.src, *pushArgs.repo)

	case untarCmd.FullCommand():
		untar(*untarArgs.file, *untarArgs.dir)

	case wipeCmd.FullCommand():
		wipe(*wipeArgs.path)

	case mkdirCmd.FullCommand():
		mkdir(*mkdirArgs.path)

	case dittoCmd.FullCommand():
		ditto(*dittoArgs.src, *dittoArgs.dst)

	case megatestCmd.FullCommand():
		megatest(*megatestArgs.src)

	case megadiffCmd.FullCommand():
		megadiff(*megadiffArgs.target, *megadiffArgs.source, *megadiffArgs.patch)

	case megapatchCmd.FullCommand():
		megapatch(*megapatchArgs.patch, *megapatchArgs.target, *megapatchArgs.output)
	}
}
