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

	dlCmd    = app.Command("dl", "Download a file (resumes if can, checks hashes)")
	pushCmd  = app.Command("push", "Upload a new version of something to itch.io")
	untarCmd = app.Command("untar", "Extract a .tar file")
)

var appArgs = struct {
	json       *bool
	quiet      *bool
	timestamps *bool
}{
	app.Flag("json", "Enable machine-readable JSON-lines output").Short('j').Bool(),
	app.Flag("quiet", "Hide progress indicators & other extra info").Short('q').Bool(),
	app.Flag("timestamps", "Prefix all output by timestamps (for logging purposes)").Bool(),
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

	switch kingpin.MustParse(cmd, err) {
	case dlCmd.FullCommand():
		dl(*dlArgs.url, *dlArgs.dest)

	case pushCmd.FullCommand():
		push(*pushArgs.src, *pushArgs.repo)

	case untarCmd.FullCommand():
		untar(*untarArgs.file, *untarArgs.dir)
	}
}
