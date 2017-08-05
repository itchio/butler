package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/go-itchio/itchfs"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pwr"
	shellquote "github.com/kballard/go-shellquote"

	"net/http"
	_ "net/http/pprof"

	"gopkg.in/alecthomas/kingpin.v2"
)

// #cgo windows LDFLAGS: -Wl,--allow-multiple-definition -static
import "C"

var (
	version       = "head" // set by command-line on CI release builds
	builtAt       = ""     // set by command-line on CI release builds
	commit        = ""     // set by command-line on CI release builds
	versionString = ""     // formatted on boot from 'version' and 'builtAt'
	app           = kingpin.New("butler", "Your happy little itch.io helper")

	scriptCmd = app.Command("script", "Run a series of butler commands").Hidden()

	dlCmd = app.Command("dl", "Download a file (resumes if can, checks hashes)").Hidden()
	cpCmd = app.Command("cp", "Copy src to dest").Hidden()

	untarCmd    = app.Command("untar", "Extract a .tar file").Hidden()
	unzipCmd    = app.Command("unzip", "Extract a .zip file").Hidden()
	indexZipCmd = app.Command("index-zip", "Generate an index for a .zip file").Hidden()
	wipeCmd     = app.Command("wipe", "Completely remove a directory (rm -rf)").Hidden()
	dittoCmd    = app.Command("ditto", "Create a mirror (incl. symlinks) of a directory into another dir (rsync -az)").Hidden()
	mkdirCmd    = app.Command("mkdir", "Create an empty directory and all required parent directories (mkdir -p)").Hidden()
	sizeofCmd   = app.Command("sizeof", "Compute the total size of a directory").Hidden()

	loginCmd  = app.Command("login", "Connect butler to your itch.io account and save credentials locally.")
	logoutCmd = app.Command("logout", "Remove saved itch.io credentials.")
	pushCmd   = app.Command("push", "Upload a new build to itch.io. See `butler help push`.")
	fetchCmd  = app.Command("fetch", "Download and extract the latest build of a channel from itch.io")
	statusCmd = app.Command("status", "Show a list of channels and the status of their latest and pending builds.")

	fileCmd  = app.Command("file", "Prints the type of a given file, and some stats about it")
	lsCmd    = app.Command("ls", "Prints the list of files, dirs and symlinks contained in a patch file, signature file, or archive")
	walkCmd  = app.Command("walk", "Finds all files in a directory").Hidden()
	cleanCmd = app.Command("clean", "Remove a bunch of files").Hidden()

	whichCmd   = app.Command("which", "Prints the path to this binary")
	versionCmd = app.Command("version", "Prints the current version of butler")
	upgradeCmd = app.Command("upgrade", "Upgrades butler to the latest version")
	ugpradeCmd = app.Command("ugprade", "Upgrades butler to the latest version (alias of upgrade for Adam)").Hidden()
	updateCmd  = app.Command("update", "Upgrades butler to the latest version (alias of upgrade)")

	signCmd   = app.Command("sign", "(Advanced) Generate a signature file for a given directory. Useful for integrity checks and remote diff generation.")
	verifyCmd = app.Command("verify", "(Advanced) Use a signature to verify the integrity of a directory")
	diffCmd   = app.Command("diff", "(Advanced) Compute the difference between two directories or .zip archives. Stores the patch in `patch.pwr`, and a signature in `patch.pwr.sig` for integrity checks and further diff.")
	applyCmd  = app.Command("apply", "(Advanced) Use a patch to patch a directory to a new version")
	healCmd   = app.Command("heal", "(Advanced) Heal a directory using a list of wounds and a heal spec")

	probeCmd = app.Command("probe", "(Advanced) Show statistics about a patch file").Hidden()

	installPrereqsCmd = app.Command("install-prereqs", "Install prerequisites from an install plan").Hidden()
	testPrereqsCmd    = app.Command("test-prereqs", "Download and install a bunch of prerequisites from their names").Hidden()
	msiInfoCmd        = app.Command("msi-info", "Show information about an MSI file").Hidden()
	msiProductInfoCmd = app.Command("msi-product-info", "Show information an installed product").Hidden()
	msiInstallCmd     = app.Command("msi-install", "Install or repair an MSI package").Hidden()
	msiUninstallCmd   = app.Command("msi-uninstall", "Uninstall an MSI package").Hidden()

	exePropsCmd  = app.Command("exeprops", "(Advanced) Gives information about an .exe file").Hidden()
	elfPropsCmd  = app.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	configureCmd = app.Command("configure", "(Advanced) Look for launchables in a directory").Hidden()
)

var (
	// DirectorySingular is whatever people call 'directories' or 'folders'
	// on their platforms. linuxers tend to prefer 'directories', so this global
	// is their punishment.
	DirectorySingular = "folder"

	// DirectoryPlural is the plural of DirectorySingular (duh)
	DirectoryPlural = "folders"
)

var appArgs = struct {
	json       *bool
	quiet      *bool
	verbose    *bool
	timestamps *bool
	noProgress *bool
	panic      *bool
	assumeYes  *bool
	beeps4Life *bool

	identity             *string
	address              *string
	compressionAlgorithm *string
	compressionQuality   *int

	maxChunkGroup *int

	cpuprofile *string
	memstats   *bool
}{
	app.Flag("json", "Enable machine-readable JSON-lines output").Hidden().Short('j').Bool(),
	app.Flag("quiet", "Hide progress indicators & other extra info").Hidden().Bool(),
	app.Flag("verbose", "Be very chatty about what's happening").Short('v').Bool(),
	app.Flag("timestamps", "Prefix all output by timestamps (for logging purposes)").Hidden().Bool(),
	app.Flag("noprogress", "Doesn't show progress bars").Hidden().Bool(),
	app.Flag("panic", "Panic on error").Hidden().Bool(),
	app.Flag("assume-yes", "Don't ask questions, just proceed (at your own risk!)").Bool(),
	app.Flag("beeps4life", "Restore historical robot bug.").Hidden().Bool(),

	app.Flag("identity", "Path to your itch.io API token").Default(defaultKeyPath()).Short('i').String(),
	app.Flag("address", "itch.io server to talk to").Default("https://itch.io").Short('a').Hidden().String(),

	app.Flag("compression", "Compression algorithm to use when writing patch or signature files").Default("brotli").Hidden().Enum("none", "brotli", "gzip"),
	app.Flag("quality", "Quality level to use when writing patch or signature files").Default("1").Short('q').Hidden().Int(),

	app.Flag("maxchunkgroup", "How many 256KB chunks butler will attempt to send in a single HTTP request").Default("64").Hidden().Int(),

	app.Flag("cpuprofile", "Write CPU profile to given file").Hidden().String(),
	app.Flag("memstats", "Print memory stats for some operations").Hidden().Bool(),
}

var scriptArgs = struct {
	file *string
}{
	scriptCmd.Arg("file", "File containing a list of butler commands, one per line, with 'butler' omitted").Required().String(),
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

var cpArgs = struct {
	src    *string
	dest   *string
	resume *bool
}{
	cpCmd.Arg("src", "File to read from").Required().String(),
	cpCmd.Arg("dest", "File to write to").Required().String(),
	cpCmd.Flag("resume", "Try to resume if dest is partially written (doesn't check existing data)").Bool(),
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
	fixPerms        *bool
}{
	pushCmd.Arg("src", "Directory to upload. May also be a zip archive (slower)").Required().String(),
	pushCmd.Arg("target", "Where to push, for example 'leafo/x-moon:win-64'. Targets are of the form project:channel, where project is username/game or game_id.").Required().String(),
	pushCmd.Flag("userversion", "A user-supplied version number that you can later query builds by").String(),
	pushCmd.Flag("userversion-file", "A file containing a user-supplied version number that you can later query builds by").String(),
	pushCmd.Flag("fix-permissions", "Detect Mac & Linux executables and adjust their permissions automatically").Default("true").Bool(),
}

var fetchArgs = struct {
	target *string
	out    *string
}{
	fetchCmd.Arg("target", "Which user/project:channel to fetch from, for example 'leafo/x-moon:win-64'. Targets are of the form project:channel where project is username/game or game_id.").Required().String(),
	fetchCmd.Arg("out", "Directory to fetch and extract build to").Required().String(),
}

var statusArgs = struct {
	target       *string
	showAllFiles *bool
}{
	statusCmd.Arg("target", "Which user/project to show the status of, for example 'leafo/x-moon'").Required().String(),
	statusCmd.Flag("show-all-files", "Show status of all files, not just archive").Bool(),
}

var untarArgs = struct {
	file *string
	dir  *string
}{
	untarCmd.Arg("file", "Path of the .tar archive to extract").Required().String(),
	untarCmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String(),
}

var unzipArgs = struct {
	file        *string
	dir         *string
	resumeFile  *string
	dryRun      *bool
	concurrency *int
}{
	unzipCmd.Arg("file", "Path of the .zip archive to extract").Required().String(),
	unzipCmd.Flag("dir", "An optional directory to which to extract files (defaults to CWD)").Default(".").Short('d').String(),
	unzipCmd.Flag("resume-file", "When given, write current progress to this file, resume from last location if it exists.").Short('f').String(),
	unzipCmd.Flag("dry-run", "Do not write anything to disk").Short('n').Bool(),
	unzipCmd.Flag("concurrency", "Number of workers to use (negative for numbers of CPUs - j)").Default("-1").Int(),
}

var indexZipArgs = struct {
	file   *string
	output *string
}{
	indexZipCmd.Arg("file", "Path of the .zip archive to index").Required().String(),
	indexZipCmd.Flag("output", "Path to write the .pzi file to").Short('o').Default("index.pzi").String(),
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

var sizeofArgs = struct {
	path *string
}{
	sizeofCmd.Arg("path", "Directory to compute the size of").Required().String(),
}

var dittoArgs = struct {
	src *string
	dst *string
}{
	dittoCmd.Arg("src", "Directory to mirror").Required().String(),
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
	wounds    *string
	heal      *string
}{
	applyCmd.Arg("patch", "Patch file (.pwr), previously generated with the `diff` command.").Required().String(),
	applyCmd.Arg("old", "Directory, archive, or empty directory (/dev/null) to patch").Required().String(),

	applyCmd.Flag("dir", "Directory to create newer files in, instead of working in-place").Short('d').String(),
	applyCmd.Flag("reverse", "When given, generates a reverse patch to allow rolling back later, along with its signature").Hidden().String(),
	applyCmd.Flag("inplace", "Apply patch directly to old directory. Required for safety").Bool(),
	applyCmd.Flag("signature", "When given, verify the integrity of touched file using the signature").String(),
	applyCmd.Flag("wounds", "When given, write wounds to this path instead of failing (exclusive with --heal)").String(),
	applyCmd.Flag("heal", "When given, heal using specified source instead of failing (exclusive with --wounds)").String(),
}

var verifyArgs = struct {
	signature *string
	dir       *string
	wounds    *string
	heal      *string
}{
	verifyCmd.Arg("signature", "Path to read signature file from").Required().String(),
	verifyCmd.Arg("dir", "Path of directory to verify").Required().String(),
	verifyCmd.Flag("wounds", "When given, writes wounds to this path").String(),
	verifyCmd.Flag("heal", "When given, heal wounds using this path").String(),
}

var signArgs = struct {
	output    *string
	signature *string
	fixPerms  *bool
}{
	signCmd.Arg("dir", "Path of directory to sign").Required().String(),
	signCmd.Arg("signature", "Path to write signature to").Required().String(),
	signCmd.Flag("fix-permissions", "Detect Mac & Linux executables and adjust their permissions automatically").Default("true").Bool(),
}

var healArgs = struct {
	dir    *string
	wounds *string
	spec   *string
}{
	healCmd.Arg("dir", "Path of directory to heal").Required().String(),
	healCmd.Arg("wounds", "Path of wounds file").Required().String(),
	healCmd.Arg("spec", "Path of spec to heal with").Required().String(),
}

var probeArgs = struct {
	patch *string
}{
	probeCmd.Arg("patch", "Path of the patch to analyze").Required().String(),
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

var walkArgs = struct {
	dir *string
}{
	walkCmd.Arg("dir", "A dir you want to walk").Required().String(),
}

var cleanArgs = struct {
	plan *string
}{
	cleanCmd.Arg("plan", "A .json plan containing a list of entries to remove").Required().String(),
}

var upgradeArgs = struct {
	head *bool
}{
	upgradeCmd.Flag("head", "Install bleeding-edge version").Bool(),
}

var ugpradeArgs = struct {
	head *bool
}{
	ugpradeCmd.Flag("head", "Install bleeding-edge version").Bool(),
}

var updateArgs = struct {
	head *bool
}{
	updateCmd.Flag("head", "Install bleeding-edge version").Bool(),
}

var installPrereqsArgs = struct {
	plan *string
	pipe *string
}{
	installPrereqsCmd.Arg("plan", "The plan to follow").Required().String(),
	installPrereqsCmd.Flag("pipe", "Named pipe where to write status updates").String(),
}

var testPrereqsArgs = struct {
	prereqs *[]string
}{
	testPrereqsCmd.Arg("prereqs", "Which prereqs to install (space-separated). Leave empty to get a list").Strings(),
}

var msiInfoArgs = struct {
	msiPath *string
}{
	msiInfoCmd.Arg("msiPath", "Path to the MSI file").Required().String(),
}

var msiProductInfoArgs = struct {
	productCode *string
}{
	msiProductInfoCmd.Arg("productCode", "The product code to print info for").Required().String(),
}

var msiInstallArgs = struct {
	msiPath *string
	logPath *string
	target  *string
}{
	msiInstallCmd.Arg("msiPath", "Path to the MSI file").Required().String(),
	msiInstallCmd.Flag("logPath", "Where to write a (very verbose) install log").String(),
	msiInstallCmd.Flag("target", "Where to install the MSI (does not work with all packages)").String(),
}

var msiUninstallArgs = struct {
	productCode *string
}{
	msiUninstallCmd.Arg("productCode", "Product code to uninstall").Required().String(),
}

var exePropsArgs = struct {
	path *string
}{
	exePropsCmd.Arg("path", "The exe to analyze").Required().String(),
}

var elfPropsArgs = struct {
	path *string
}{
	elfPropsCmd.Arg("path", "The ELF binary to analyze").Required().String(),
}

var configureArgs = struct {
	path       *string
	showSpell  *bool
	osFilter   *string
	archFilter *string
	noFilter   *bool
}{
	configureCmd.Arg("path", "The directory to configure").Required().String(),
	configureCmd.Flag("show-spell", "Show spell for all targets").Bool(),
	configureCmd.Flag("os-filter", "OS filter").Default(runtime.GOOS).String(),
	configureCmd.Flag("arch-filter", "Architecture filter").Default(runtime.GOARCH).String(),
	configureCmd.Flag("no-filter", "Do not filter at all").Bool(),
}

func must(err error) {
	if err != nil {
		switch err := err.(type) {
		case *errors.Error:
			comm.Die(err.ErrorStack())
		default:
			comm.Die(err.Error())
		}
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
	if runtime.GOOS == "linux" {
		// *long sigh*
		DirectorySingular = "directory"
		DirectoryPlural = "directories"
	}

	doMain(os.Args[1:])
}

func doMain(args []string) {
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Flag("ignore", "Glob patterns of files to ignore when diffing").StringsVar(&filtering.IgnoredPaths)

	app.HelpFlag.Short('h')
	if builtAt != "" {
		epoch, err := strconv.ParseInt(builtAt, 10, 64)
		must(err)
		versionString = fmt.Sprintf("%s, built on %s", version, time.Unix(epoch, 0).Format("Jan _2 2006 @ 15:04:05"))
	} else {
		versionString = fmt.Sprintf("%s, no build date", version)
	}
	if commit != "" {
		versionString = fmt.Sprintf("%s, ref %s", versionString, commit)
	}
	app.Version(versionString)
	app.VersionFlag.Short('V')
	app.Author("Amos Wenger <amos@itch.io>")

	cmd, err := app.Parse(args)
	if err != nil {
		ctx, _ := app.ParseContext(os.Args[1:])
		if ctx != nil {
			app.FatalUsageContext(ctx, "%s\n", err.Error())
		} else {
			app.FatalUsage("%s\n", err.Error())
		}
	}

	if *appArgs.timestamps {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}
	log.SetOutput(os.Stdout)

	eos.RegisterHandler(&itchfs.ItchFS{
		ItchServer: *appArgs.address,
	})

	if *appArgs.quiet {
		*appArgs.noProgress = true
		*appArgs.verbose = false
	}

	if !isTerminal() {
		*appArgs.noProgress = true
	}
	comm.Configure(*appArgs.noProgress, *appArgs.quiet, *appArgs.verbose, *appArgs.json, *appArgs.panic, *appArgs.assumeYes, *appArgs.beeps4Life)
	if !isTerminal() {
		comm.Debug("Not a terminal, disabling progress indicator")
	}

	setupHTTPDebug()

	if *appArgs.cpuprofile != "" {
		f, err := os.Create(*appArgs.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	switch kingpin.MustParse(cmd, err) {
	case scriptCmd.FullCommand():
		script(*scriptArgs.file)

	case dlCmd.FullCommand():
		dl(*dlArgs.url, *dlArgs.dest)

	case cpCmd.FullCommand():
		cp(*cpArgs.src, *cpArgs.dest, *cpArgs.resume)

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
			push(*pushArgs.src, *pushArgs.target, userVersion, *pushArgs.fixPerms)
		}

	case fetchCmd.FullCommand():
		fetch(*fetchArgs.target, *fetchArgs.out)

	case statusCmd.FullCommand():
		status(*statusArgs.target, *statusArgs.showAllFiles)

	case untarCmd.FullCommand():
		untar(*untarArgs.file, *untarArgs.dir)

	case unzipCmd.FullCommand():
		unzip(*unzipArgs.file, *unzipArgs.dir, *unzipArgs.resumeFile, *unzipArgs.dryRun, *unzipArgs.concurrency)

	case indexZipCmd.FullCommand():
		indexZip(*indexZipArgs.file, *indexZipArgs.output)

	case wipeCmd.FullCommand():
		wipe(*wipeArgs.path)

	case mkdirCmd.FullCommand():
		mkdir(*mkdirArgs.path)

	case dittoCmd.FullCommand():
		ditto(*dittoArgs.src, *dittoArgs.dst)

	case sizeofCmd.FullCommand():
		sizeof(*sizeofArgs.path)

	case diffCmd.FullCommand():
		diff(*diffArgs.old, *diffArgs.new, *diffArgs.patch, butlerCompressionSettings())

	case applyCmd.FullCommand():
		apply(*applyArgs.patch, *applyArgs.old, *applyArgs.dir, *applyArgs.inplace, *applyArgs.signature, *applyArgs.wounds, *applyArgs.heal)

	case verifyCmd.FullCommand():
		verify(*verifyArgs.signature, *verifyArgs.dir, *verifyArgs.wounds, *verifyArgs.heal)

	case signCmd.FullCommand():
		sign(*signArgs.output, *signArgs.signature, butlerCompressionSettings(), *signArgs.fixPerms)

	case healCmd.FullCommand():
		heal(*healArgs.dir, *healArgs.wounds, *healArgs.spec)

	case probeCmd.FullCommand():
		probe(*probeArgs.patch)

	case whichCmd.FullCommand():
		which()

	case versionCmd.FullCommand():
		log.Println(versionString)

	case fileCmd.FullCommand():
		file(*fileArgs.file)

	case lsCmd.FullCommand():
		ls(*lsArgs.file)

	case walkCmd.FullCommand():
		walk(*walkArgs.dir)

	case cleanCmd.FullCommand():
		clean(*cleanArgs.plan)

	case upgradeCmd.FullCommand():
		upgrade(*upgradeArgs.head)

	case ugpradeCmd.FullCommand():
		upgrade(*upgradeArgs.head)

	case updateCmd.FullCommand():
		upgrade(*updateArgs.head)

	case installPrereqsCmd.FullCommand():
		installPrereqs(*installPrereqsArgs.plan, *installPrereqsArgs.pipe)

	case testPrereqsCmd.FullCommand():
		testPrereqs(*testPrereqsArgs.prereqs)

	case msiInfoCmd.FullCommand():
		msiInfo(*msiInfoArgs.msiPath)

	case msiProductInfoCmd.FullCommand():
		msiProductInfo(*msiProductInfoArgs.productCode)

	case msiInstallCmd.FullCommand():
		msiInstall(*msiInstallArgs.msiPath, *msiInstallArgs.logPath, *msiInstallArgs.target)

	case msiUninstallCmd.FullCommand():
		msiUninstall(*msiUninstallArgs.productCode)

	case exePropsCmd.FullCommand():
		exeProps(*exePropsArgs.path)

	case elfPropsCmd.FullCommand():
		elfProps(*elfPropsArgs.path)

	case configureCmd.FullCommand():
		configure(*configureArgs.path, *configureArgs.showSpell, *configureArgs.osFilter, *configureArgs.archFilter, *configureArgs.noFilter)
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

func script(scriptPath string) {
	must(doScript(scriptPath))
}

func doScript(scriptPath string) error {
	scriptReader, err := os.Open(scriptPath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(scriptReader)
	comm.Logf("Running commands in script %s", scriptPath)

	for scanner.Scan() {
		argsString := scanner.Text()
		comm.Opf("butler %s", argsString)

		args, err := shellquote.Split(argsString)
		if err != nil {
			return fmt.Errorf("While parsing `%s`: %s", argsString, err.Error())
		}
		doMain(args)
	}
	return nil
}
