package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/itchio/damage"
	"github.com/itchio/damage/hdiutil"
	"github.com/itchio/wharf/state"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app     = kingpin.New("damage", "Your devilish little DMG helper")
	verbose = app.Flag("verbose", "Enable verbose output").Short('v').Bool()

	slaCmd  = app.Command("sla", "Show SLA (software license agreement) for dmg")
	slaFile = slaCmd.Arg("file", "The .dmg file to print the SLA for").ExistingFile()

	derezCmd  = app.Command("derez", "Print resources from a DMG file")
	derezFile = derezCmd.Arg("file", "The .dmg file to print resources from").ExistingFile()

	infoCmd  = app.Command("info", "Print information about a DMG file")
	infoFile = infoCmd.Arg("file", "The .dmg file to analyze").ExistingFile()
	infoLong = infoCmd.Flag("long", "Show all info").Bool()

	mountCmd  = app.Command("mount", "Mount a DMG file to a local folder (ignoring the SLA)")
	mountFile = mountCmd.Arg("file", "The .dmg file to mount").ExistingFile()
	mountDir  = mountCmd.Flag("dir", "Where to mount the dmg").Short('d').ExistingDir()

	unmountCmd = app.Command("unmount", "Unmount a DMG file mounted to a local folder")
	unmountDir = unmountCmd.Arg("file", "The directory to unmount").ExistingDir()

	consumer *state.Consumer
	host     hdiutil.Host
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	app.UsageTemplate(kingpin.CompactUsageTemplate)

	app.HelpFlag.Short('h')
	app.Version("head")
	app.VersionFlag.Short('V')
	app.Author("Amos Wenger <amos@itch.io>")

	consumer = &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			if lvl == "debug" && !*verbose {
				return
			}
			log.Printf("[%s] %s", lvl, msg)
		},
	}

	args := os.Args[1:]
	cmd, err := app.Parse(args)
	if err != nil {
		ctx, _ := app.ParseContext(os.Args[1:])
		if ctx != nil {
			app.FatalUsageContext(ctx, "%s\n", err.Error())
		} else {
			app.FatalUsage("%s\n", err.Error())
		}
	}

	host = hdiutil.NewHost(consumer)
	if *verbose {
		log.Printf("Running in verbose mode")
		host.SetDump(spew.Dump)
	}

	fullCmd := kingpin.MustParse(cmd, err)
	switch fullCmd {
	case infoCmd.FullCommand():
		info()
	case derezCmd.FullCommand():
		derez()
	case slaCmd.FullCommand():
		sla()
	case mountCmd.FullCommand():
		mount()
	case unmountCmd.FullCommand():
		unmount()
	}
}

func info() {
	file := *infoFile

	info, err := damage.GetDiskInfo(host, file)
	must(err)

	if *infoLong {
		jsonDump(info)
	} else {
		log.Printf("============================")
		log.Printf("%s", file)
		log.Printf("----------------------------")
		log.Printf("%s", info)
		log.Printf("============================")
	}
}

func derez() {
	file := *derezFile

	rez, err := damage.GetUDIFResources(host, file)
	must(err)

	if *infoLong {
		jsonDump(info)
	} else {
		log.Printf("============================")
		log.Printf("%s", file)
		log.Printf("----------------------------")
		log.Printf("%s", rez)
		log.Printf("============================")
	}
}

func sla() {
	file := *slaFile

	info, err := damage.GetDiskInfo(host, file)
	must(err)

	if !info.Properties.SoftwareLicenseAgreement {
		log.Printf("%s: no SLA", file)
		return
	}

	rez, err := damage.GetUDIFResources(host, file)
	must(err)

	sla, err := damage.GetDefaultSLA(rez)
	must(err)

	log.Printf("%s: %s SLA", file, sla.Language)
	text := strings.Replace(sla.Text, "\r", "\n", -1)
	lines := strings.Split(text, "\n")
	maxColumns := 80
	for _, line := range lines {
		for len(line) > maxColumns {
			log.Printf("%s", line[:maxColumns])
			line = line[maxColumns:]
		}
		log.Printf("%s", line)
	}
}

func mount() {
	file := *mountFile
	dir := *mountDir

	res, err := damage.Mount(host, file, dir)
	must(err)

	log.Printf("%s: mounted", file)
	for _, entity := range res.SystemEntities {
		if entity.MountPoint != "" {
			log.Printf("%s: %s (%s) (%s)", entity.MountPoint, entity.VolumeKind, entity.ContentHint, entity.DevEntry)
		}
	}
}

func unmount() {
	dir := *unmountDir

	err := damage.Unmount(host, dir)
	must(err)

	log.Printf("%s: unmounted", dir)
}

func jsonDump(v interface{}) {
	out, err := json.MarshalIndent(v, "", "  ")
	must(err)

	log.Print(string(out))
}

func must(err error) {
	if err != nil {
		panic(fmt.Sprintf("fatal error: %+v", err))
	}
}
