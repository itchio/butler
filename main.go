package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/apply"
	"github.com/itchio/butler/cmd/cave"
	"github.com/itchio/butler/cmd/clean"
	"github.com/itchio/butler/cmd/cp"
	"github.com/itchio/butler/cmd/diff"
	"github.com/itchio/butler/cmd/ditto"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/cmd/fetch"
	"github.com/itchio/butler/cmd/file"
	"github.com/itchio/butler/cmd/heal"
	"github.com/itchio/butler/cmd/indexzip"
	"github.com/itchio/butler/cmd/login"
	"github.com/itchio/butler/cmd/logout"
	"github.com/itchio/butler/cmd/ls"
	"github.com/itchio/butler/cmd/mkdir"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/cmd/pipe"
	"github.com/itchio/butler/cmd/prereqs"
	"github.com/itchio/butler/cmd/probe"
	"github.com/itchio/butler/cmd/sign"
	"github.com/itchio/butler/cmd/sizeof"
	"github.com/itchio/butler/cmd/status"
	"github.com/itchio/butler/cmd/untar"
	"github.com/itchio/butler/cmd/unzip"
	"github.com/itchio/butler/cmd/upgrade"
	"github.com/itchio/butler/cmd/verify"
	"github.com/itchio/butler/cmd/version"
	"github.com/itchio/butler/cmd/walk"
	"github.com/itchio/butler/cmd/which"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
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
	butlerVersion       = "head" // set by command-line on CI release builds
	butlerBuiltAt       = ""     // set by command-line on CI release builds
	butlerCommit        = ""     // set by command-line on CI release builds
	butlerVersionString = ""     // formatted on boot from 'version' and 'builtAt'
	app                 = kingpin.New("butler", "Your happy little itch.io helper")

	scriptCmd = app.Command("script", "Run a series of butler commands").Hidden()

	exePropsCmd  = app.Command("exeprops", "(Advanced) Gives information about an .exe file").Hidden()
	elfPropsCmd  = app.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	configureCmd = app.Command("configure", "(Advanced) Look for launchables in a directory").Hidden()
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

	cpuprofile *string
	memstats   *bool
	elevate    *bool
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

	app.Flag("compression", "Compression algorithm to use when writing patch or signature files").Default("brotli").Hidden().Enum("none", "brotli", "gzip", "zstd"),
	app.Flag("quality", "Quality level to use when writing patch or signature files").Default("1").Short('q').Hidden().Int(),

	app.Flag("cpuprofile", "Write CPU profile to given file").Hidden().String(),
	app.Flag("memstats", "Print memory stats for some operations").Hidden().Bool(),

	app.Flag("elevate", "Run butler as administrator").Hidden().Bool(),
}

var scriptArgs = struct {
	file *string
}{
	scriptCmd.Arg("file", "File containing a list of butler commands, one per line, with 'butler' omitted").Required().String(),
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

func butlerCompressionSettings() pwr.CompressionSettings {
	var algo pwr.CompressionAlgorithm

	switch *appArgs.compressionAlgorithm {
	case "none":
		algo = pwr.CompressionAlgorithm_NONE
	case "brotli":
		algo = pwr.CompressionAlgorithm_BROTLI
	case "gzip":
		algo = pwr.CompressionAlgorithm_GZIP
	case "zstd":
		algo = pwr.CompressionAlgorithm_ZSTD
	default:
		panic(fmt.Errorf("Unknown compression algorithm: %s", algo))
	}

	return pwr.CompressionSettings{
		Algorithm: algo,
		Quality:   int32(*appArgs.compressionQuality),
	}
}

func main() {
	doMain(os.Args[1:])
}

func doMain(args []string) {
	ctx := mansion.NewContext(app)

	///////////////////////////
	// Command register start
	///////////////////////////

	version.Register(ctx)
	which.Register(ctx)

	login.Register(ctx)
	logout.Register(ctx)
	upgrade.Register(ctx)

	dl.Register(ctx)
	cp.Register(ctx)
	ls.Register(ctx)
	wipe.Register(ctx)
	sizeof.Register(ctx)
	mkdir.Register(ctx)
	ditto.Register(ctx)
	file.Register(ctx)
	probe.Register(ctx)

	clean.Register(ctx)
	walk.Register(ctx)

	sign.Register(ctx)
	diff.Register(ctx)
	apply.Register(ctx)
	verify.Register(ctx)
	heal.Register(ctx)

	status.Register(ctx)
	fetch.Register(ctx)

	msi.Register(ctx)
	prereqs.Register(ctx)

	unzip.Register(ctx)
	untar.Register(ctx)
	indexzip.Register(ctx)

	pipe.Register(ctx)
	elevate.Register(ctx)

	cave.Register(ctx)

	///////////////////////////
	// Command register end
	///////////////////////////

	app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Flag("ignore", "Glob patterns of files to ignore when diffing").StringsVar(&filtering.IgnoredPaths)

	app.HelpFlag.Short('h')
	if butlerBuiltAt != "" {
		epoch, err := strconv.ParseInt(butlerBuiltAt, 10, 64)
		must(err)
		butlerVersionString = fmt.Sprintf("%s, built on %s", butlerVersion, time.Unix(epoch, 0).Format("Jan _2 2006 @ 15:04:05"))
	} else {
		butlerVersionString = fmt.Sprintf("%s, no build date", butlerVersion)
	}
	if butlerCommit != "" {
		butlerVersionString = fmt.Sprintf("%s, ref %s", butlerVersionString, butlerCommit)
	}
	app.Version(butlerVersionString)
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

	if *appArgs.elevate {
		var cmdLine []string

		butlerExe, err := os.Executable()
		must(err)

		cmdLine = append(cmdLine, butlerExe)

		pastAppArgs := false
		for _, arg := range args {
			if !pastAppArgs {
				if arg == "--elevate" {
					// skip --elevate, otherwise we're getting into a loop :)
					continue
				} else if arg == "--" {
					pastAppArgs = true
				}
			}

			cmdLine = append(cmdLine, arg)
		}
		elevate.Do(cmdLine)
		return
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

	fullCmd := kingpin.MustParse(cmd, err)

	ctx.Identity = *appArgs.identity
	ctx.Address = *appArgs.address
	ctx.VersionString = butlerVersionString
	ctx.Version = butlerVersion
	ctx.Quiet = *appArgs.quiet
	ctx.Verbose = *appArgs.verbose
	ctx.CompressionAlgorithm = *appArgs.compressionAlgorithm
	ctx.CompressionQuality = *appArgs.compressionQuality

	switch fullCmd {
	case scriptCmd.FullCommand():
		script(*scriptArgs.file)

	case exePropsCmd.FullCommand():
		exeProps(*exePropsArgs.path)

	case elfPropsCmd.FullCommand():
		elfProps(*elfPropsArgs.path)

	case configureCmd.FullCommand():
		configure(*configureArgs.path, *configureArgs.showSpell, *configureArgs.osFilter, *configureArgs.archFilter, *configureArgs.noFilter)

	default:
		do := ctx.Commands[fullCmd]
		if do != nil {
			do(ctx)
		} else {
			comm.Dief("Unknown command: %s", fullCmd)
		}
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
