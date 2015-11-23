package main

import (
	"fmt"
	"log"
	"os"

	"github.com/itchio/butler/bio"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	version    = "head" // set by command-line on CI release builds
	app        = kingpin.New("butler", "Your very own itch.io helper")
	jsonOutput = app.Flag("json", "Enable machine-readable JSON-lines output").Short('j').Bool()
	quiet      = app.Flag("quiet", "Hide progress indicators & other extra info").Short('q').Bool()
	timestamps = app.Flag("timestamps", "Prefix all output by timestamps (for logging purposes)").Bool()

	dlCmd  = app.Command("dl", "Download a file (resumes if can, checks hashes)")
	dlUrl  = dlCmd.Arg("url", "Address to download from").Required().String()
	dlDest = dlCmd.Arg("dest", "File to write downloaded data to").Required().String()

	pushCmd      = app.Command("push", "Upload a new version of something to itch.io")
	pushIdentity = pushCmd.Flag("identity", "Path to the private key used for public key authentication.").Default(fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".ssh/id_rsa")).Short('i').ExistingFile()
	pushAddress  = pushCmd.Flag("address", "Specify wharf address (advanced)").Default("wharf.itch.zone").Short('a').Hidden().String()
	pushSrc      = pushCmd.Arg("src", "Directory or zip archive to upload, e.g.").Required().ExistingFileOrDir()
	pushRepo     = pushCmd.Arg("repo", "Repository to push to, e.g. leafo/xmoon:win64").Required().String()
)

func main() {
	app.HelpFlag.Short('h')
	app.Version(version)
	app.VersionFlag.Short('V')

	cmd, err := app.Parse(os.Args[1:])
	bio.JsonOutput = *jsonOutput
	bio.Quiet = *quiet
	if !*timestamps {
		log.SetFlags(0)
	}

	switch kingpin.MustParse(cmd, err) {
	case dlCmd.FullCommand():
		dl(*dlUrl, *dlDest)

	case pushCmd.FullCommand():
		push(*pushSrc, *pushRepo)
	}
}
