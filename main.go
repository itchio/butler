package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/wharf/pwr"

	"net/http"
	_ "net/http/pprof"

	"gopkg.in/alecthomas/kingpin.v2"
)

// #cgo windows LDFLAGS: -Wl,--allow-multiple-definition -static
import "C"

var (
	version       = "head" // set by command-line on CI release builds
	builtAt       = ""     // set by command-line on CI release builds
	versionString = ""     // formatted on boot from 'version' and 'builtAt'
	app           = kingpin.New("butler", "Your happy little itch.io helper")

	dlCmd = app.Command("dl", "Download a file (resumes if can, checks hashes)").Hidden()

	untarCmd = app.Command("untar", "Extract a .tar file").Hidden()
	wipeCmd  = app.Command("wipe", "Completely remove a directory (rm -rf)").Hidden()
	dittoCmd = app.Command("ditto", "Create a mirror (incl. symlinks) of a directory into another dir (rsync -az)").Hidden()
	mkdirCmd = app.Command("mkdir", "Create an empty directory and all required parent directories (mkdir -p)").Hidden()

	loginCmd  = app.Command("login", "Connect butler to your itch.io account and save credentials locally.")
	logoutCmd = app.Command("logout", "Remove saved itch.io credentials.")
	pushCmd   = app.Command("push", "Upload a new build to itch.io. See `butler help push`.")
	fetchCmd  = app.Command("fetch", "Download and extract the latest build of a channel from itch.io")
	statusCmd = app.Command("status", "Show a list of channels and the status of their latest and pending builds.")

	fileCmd = app.Command("file", "Prints the type of a given file, and some stats about it")
	lsCmd   = app.Command("ls", "Prints the list of files, dirs and symlinks contained in a patch file, signature file, or archive")

	whichCmd   = app.Command("which", "Prints the path to this binary")
	upgradeCmd = app.Command("upgrade", "Upgrades butler to the latest version")

	signCmd   = app.Command("sign", "(Advanced) Generate a signature file for a given directory. Useful for integrity checks and remote diff generation.")
	verifyCmd = app.Command("verify", "(Advanced) Use a signature to verify the integrity of a directory")
	diffCmd   = app.Command("diff", "(Advanced) Compute the difference between two directories or .zip archives. Stores the patch in `patch.pwr`, and a signature in `patch.pwr.sig` for integrity checks and further diff.")
	applyCmd  = app.Command("apply", "(Advanced) Use a patch to patch a directory to a new version")
)

var appArgs = struct {
	json       *bool
	quiet      *bool
	verbose    *bool
	timestamps *bool
	noProgress *bool
	panic      *bool

	identity             *string
	address              *string
	compressionAlgorithm *string
	compressionQuality   *int
}{
	app.Flag("json", "Enable machine-readable JSON-lines output").Hidden().Short('j').Bool(),
	app.Flag("quiet", "Hide progress indicators & other extra info").Hidden().Bool(),
	app.Flag("verbose", "Be very chatty about what's happening").Short('v').Bool(),
	app.Flag("timestamps", "Prefix all output by timestamps (for logging purposes)").Hidden().Bool(),
	app.Flag("noprogress", "Doesn't show progress bars").Hidden().Bool(),
	app.Flag("panic", "Panic on error").Hidden().Bool(),

	app.Flag("identity", "Path to your itch.io API token").Default(defaultKeyPath()).Short('i').String(),
	app.Flag("address", "itch.io server to talk to").Default("https://itch.io").Short('a').Hidden().String(),

	app.Flag("compression", "Compression algorithm to use when writing patch or signature files").Default("brotli").Enum("none", "brotli", "gzip"),
	app.Flag("quality", "Quality level to use when writing patch or signature files").Default("1").Short('q').Int(),
}

var dlArgs = struct {
	url  *string
	dest *string

	thorough *bool
}{
	dlCmd.Arg("url", "Address to download from").Required().String(),
	dlCmd.Arg("dest", "File to write downloaded data to").Required().String(),

	dlCmd.Flag("thorough", "Check all available hashes").Bool(),
}

func defaultKeyPath() string {
	configPath := os.Getenv("XDG_CONFIG_PATH")
	if configPath == "" {
		dir := ".config/itch"
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}

		if runtime.GOOS == "darwin" {
			home = path.Join(home, "Library", "Application Support")
			dir = "itch"
		}
		configPath = filepath.FromSlash(path.Join(home, dir, "butler_creds"))
	}
	return configPath
}

var pushArgs = struct {
	src             *string
	target          *string
	userVersion     *string
	userVersionFile *string
}{
	pushCmd.Arg("src", "Directory to upload. May also be a zip archive (slower)").Required().String(),
	pushCmd.Arg("target", "Where to push, for example 'leafo/xmoon:win-64'. Targets are of the form project:channel, where project is username/game or game_id.").Required().String(),
	pushCmd.Flag("userversion", "A user-supplied version number that you can later query builds by").String(),
	pushCmd.Flag("userversion-file", "A file containing a user-supplied version number that you can later query builds by").String(),
}

var fetchArgs = struct {
	target *string
	out    *string
}{
	fetchCmd.Arg("target", "Which user/project:channel to fetch from, for example 'leafo/xmoon:win-64'. Targets are of the form project:channel where project is username/game or game_id.").Required().String(),
	fetchCmd.Arg("out", "Directory to fetch and extract build to").Required().String(),
}

var statusArgs = struct {
	target *string
}{
	statusCmd.Arg("target", "Which user/project:channel to fetch from, for example 'leafo/xmoon:win-64'. Targets are of the form project:channel where project is username/game or game_id.").Required().String(),
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
	dittoCmd.Arg("src", "Directory to mirror").Required().ExistingFileOrDir(),
	dittoCmd.Arg("dst", "Path where to create a mirror").Required().String(),
}

var diffArgs = struct {
	old   *string
	new   *string
	patch *string

	verify *bool
}{
	diffCmd.Arg("old", "Directory or .zip archive (slower) with older files, or signature file generated from old directory.").Required().String(),
	diffCmd.Arg("new", "Directory or .zip archive (slower) with newer files").Required().String(),
	diffCmd.Arg("patch", "Path to write the patch file (recommended extension is `.pwr`) The signature file will be written to the same path, with .sig added to the end.").Default("patch.pwr").String(),

	diffCmd.Flag("verify", "Make sure generated patch applies cleanly by applying it (slower)").Bool(),
}

var applyArgs = struct {
	patch *string
	old   *string

	dir       *string
	reverse   *string
	inplace   *bool
	signature *string
}{
	applyCmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().ExistingFileOrDir(),
	applyCmd.Arg("old", "Directory to patch").Required().ExistingFileOrDir(),

	applyCmd.Flag("dir", "Directory to create newer files in, instead of working in-place").Short('d').String(),
	applyCmd.Flag("reverse", "When given, generates a reverse patch to allow rolling back later, along with its signature").Hidden().String(),
	applyCmd.Flag("inplace", "Apply patch directly to old directory. Required for safety").Bool(),
	applyCmd.Flag("signature", "When given, verify the integrity of touched file using the signature").String(),
}

var verifyArgs = struct {
	signature *string
	output    *string
}{
	verifyCmd.Arg("signature", "Path to read signature file from").Required().String(),
	verifyCmd.Arg("dir", "Path of directory to verify").Required().String(),
}

var signArgs = struct {
	output    *string
	signature *string
}{
	signCmd.Arg("dir", "Path of directory to sign").Required().String(),
	signCmd.Arg("signature", "Path to write signature to").Required().String(),
}

var fileArgs = struct {
	file *string
}{
	fileCmd.Arg("file", "A file you'd like to identify").Required().String(),
}

var lsArgs = struct {
	file *string
}{
	lsCmd.Arg("file", "A file you'd like to list the contents of").Required().String(),
}

var upgradeArgs = struct {
	assumeYes *bool
}{
	upgradeCmd.Flag("assume-yes", "Don't ask questions, just proceed").Bool(),
}

func must(err error) {
	if err != nil {
		comm.Die(err.Error())
	}
}

func userAgent() string {
	return fmt.Sprintf("butler/%s", versionString)
}

func butlerCompressionSettings() pwr.CompressionSettings {
	var algo pwr.CompressionAlgorithm

	switch *appArgs.compressionAlgorithm {
	case "none":
		algo = pwr.CompressionAlgorithm_NONE
	case "brotli":
		algo = pwr.CompressionAlgorithm_BROTLI
	case "gzip":
		algo = pwr.CompressionAlgorithm_GZIP
	default:
		panic(fmt.Errorf("Unknown compression algorithm: %s", algo))
	}

	return pwr.CompressionSettings{
		Algorithm: algo,
		Quality:   int32(*appArgs.compressionQuality),
	}
}

func main() {
	app.Flag("ignore", "Glob patterns of files to ignore when diffing").StringsVar(&ignoredPaths)

	app.HelpFlag.Short('h')
	if builtAt != "" {
		epoch, err := strconv.ParseInt(builtAt, 10, 64)
		must(err)
		versionString = fmt.Sprintf("%s, built on %s", version, time.Unix(epoch, 0).Format("Jan _2 2006 @ 15:04:05"))
	} else {
		versionString = fmt.Sprintf("%s, no build date", version)
	}
	app.Version(versionString)
	app.VersionFlag.Short('V')
	app.Author("Amos Wenger <amos@itch.io>")

	cmd, err := app.Parse(os.Args[1:])
	if *appArgs.timestamps {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}

	if *appArgs.quiet {
		*appArgs.noProgress = true
		*appArgs.verbose = false
	}

	if !isTerminal() {
		*appArgs.noProgress = true
	}
	comm.Configure(*appArgs.noProgress, *appArgs.quiet, *appArgs.verbose, *appArgs.json, *appArgs.panic)
	if !isTerminal() {
		comm.Debug("Not a terminal, disabling progress indicator")
	}

	setupHTTPDebug()

	switch kingpin.MustParse(cmd, err) {
	case dlCmd.FullCommand():
		dl(*dlArgs.url, *dlArgs.dest)

	case loginCmd.FullCommand():
		login()

	case logoutCmd.FullCommand():
		logout()

	case pushCmd.FullCommand():
		{
			userVersion := *pushArgs.userVersion
			if userVersion == "" && *pushArgs.userVersionFile != "" {
				buf, err := ioutil.ReadFile(*pushArgs.userVersionFile)
				must(err)
				userVersion = strings.TrimSpace(string(buf))
				if strings.ContainsAny(userVersion, "\r\n") {
					must(fmt.Errorf("%s contains line breaks, refusing to use as userversion", *pushArgs.userVersionFile))
				}
			}
			push(*pushArgs.src, *pushArgs.target, userVersion)
		}

	case fetchCmd.FullCommand():
		fetch(*fetchArgs.target, *fetchArgs.out)

	case statusCmd.FullCommand():
		status(*statusArgs.target)

	case untarCmd.FullCommand():
		untar(*untarArgs.file, *untarArgs.dir)

	case wipeCmd.FullCommand():
		wipe(*wipeArgs.path)

	case mkdirCmd.FullCommand():
		mkdir(*mkdirArgs.path)

	case dittoCmd.FullCommand():
		ditto(*dittoArgs.src, *dittoArgs.dst)

	case diffCmd.FullCommand():
		diff(*diffArgs.old, *diffArgs.new, *diffArgs.patch, butlerCompressionSettings())

	case applyCmd.FullCommand():
		apply(*applyArgs.patch, *applyArgs.old, *applyArgs.dir, *applyArgs.inplace, *applyArgs.signature)

	case verifyCmd.FullCommand():
		verify(*verifyArgs.signature, *verifyArgs.output)

	case signCmd.FullCommand():
		sign(*signArgs.output, *signArgs.signature, butlerCompressionSettings())

	case whichCmd.FullCommand():
		which()

	case fileCmd.FullCommand():
		file(*fileArgs.file)

	case lsCmd.FullCommand():
		ls(*lsArgs.file)

	case upgradeCmd.FullCommand():
		upgrade(*upgradeArgs.assumeYes)
	}
}

func setupHTTPDebug() {
	debugPort := os.Getenv("BUTLER_DEBUG_PORT")

	if debugPort == "" {
		return
	}

	addr := fmt.Sprintf("localhost:%s", debugPort)
	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			comm.Logf("http debug error: %s", err.Error())
		}
	}()
	comm.Logf("serving pprof debug interface on %s", addr)
}
