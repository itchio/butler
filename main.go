package main

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/efarrer/iothrottler"

	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"

	"github.com/itchio/go-itchio/itchfs"

	"github.com/itchio/headway/united"

	"github.com/itchio/httpkit/timeout"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"

	shellquote "github.com/kballard/go-shellquote"

	"net/http"
	_ "net/http/pprof"

	"gopkg.in/alecthomas/kingpin.v2"
)

import "C"

var (
	app                 = kingpin.New("butler", "Your happy little itch.io helper")

	scriptCmd = app.Command("script", "Run a series of butler commands").Hidden()
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
	userAgentAddition    *string
	dbPath               *string
	compressionAlgorithm *string
	compressionQuality   *int

	cpuprofile *string
	memstats   *bool
	elevate    *bool

	throttle        *int64
	simulateOffline *bool
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
	app.Flag("address", "itch.io server to talk to").Default("https://api.itch.io").Short('a').Hidden().String(),
	app.Flag("user-agent", "string to include in user-agent for all http requests").Default("").Hidden().String(),
	app.Flag("dbpath", "Path of the sqlite database path to use (for butlerd)").Default("").Hidden().String(),

	app.Flag("compression", "Compression algorithm to use when writing patch or signature files").Default("brotli").Hidden().Enum("none", "brotli", "gzip"),
	app.Flag("quality", "Quality level to use when writing patch or signature files").Default("1").Short('q').Hidden().Int(),

	app.Flag("cpuprofile", "Write CPU profile to given file").Hidden().String(),
	app.Flag("memstats", "Print memory stats for some operations").Hidden().Bool(),

	app.Flag("elevate", "Run butler as administrator").Hidden().Bool(),

	app.Flag("throttle", "Use less than 'throttle' Kbps (kilobits per second) of bandwidth").Hidden().Default("-1").Int64(),
	app.Flag("simulateOffline", "Simulate offline").Hidden().Default("false").Bool(),
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

func must(err error) {
	if err != nil {
		if *appArgs.verbose || *appArgs.json {
			comm.Dief("%+v", err)
		} else {
			comm.Dief("%v", err)
		}
	}
}

func main() {
	doMain(os.Args[1:])
}

func doMain(args []string) {
	rand.Seed(time.Now().UnixNano())

	ctx := mansion.NewContext(app)
	registerCommands(ctx)

	app.UsageTemplate(kingpin.CompactUsageTemplate)
	app.Flag("ignore", "Glob patterns of files to ignore when pushing or diffing").StringsVar(&filtering.IgnoredPaths)

	app.HelpFlag.Short('h')
	app.Version(buildinfo.VersionString)
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

	// we need to handle self-elevate real soon
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
		must(elevate.Do(cmdLine))
		return
	}

	if *appArgs.timestamps {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		log.SetFlags(0)
	}
	log.SetOutput(os.Stdout)

	if *appArgs.quiet {
		*appArgs.noProgress = true
		*appArgs.verbose = false
	}

	if !mansion.IsTerminal() {
		*appArgs.noProgress = true
	}
	comm.Configure(*appArgs.noProgress, *appArgs.quiet, *appArgs.verbose, *appArgs.json, *appArgs.panic, *appArgs.assumeYes, *appArgs.beeps4Life)
	if !mansion.IsTerminal() {
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

	if *appArgs.throttle > 0 {
		throttle := *appArgs.throttle
		bwKiloBytes := throttle / 8 * 1024
		comm.Logf("Throttling to %s/s bandwidth", united.FormatBytes(bwKiloBytes))
		timeout.ThrottlerPool.SetBandwidth(iothrottler.Bandwidth(throttle) * iothrottler.Kbps)
	}

	if *appArgs.simulateOffline {
		timeout.SetSimulateOffline(true)
	}

	fullCmd := kingpin.MustParse(cmd, err)

	ctx.Identity = *appArgs.identity
	ctx.SetAddress(*appArgs.address)
	ctx.UserAgentAddition = *appArgs.userAgentAddition
	ctx.DBPath = *appArgs.dbPath
	ctx.Quiet = *appArgs.quiet
	ctx.Verbose = *appArgs.verbose
	ctx.JSON = *appArgs.json
	ctx.CompressionAlgorithm = *appArgs.compressionAlgorithm
	ctx.CompressionQuality = *appArgs.compressionQuality

	// set up eos
	{
		eos.RegisterHandler(&itchfs.ItchFS{
			ItchServer: *appArgs.address,
			UserAgent:  ctx.UserAgent(),
		})
		option.SetDefaultConsumer(comm.NewStateConsumer())
		option.SetDefaultHTTPClient(ctx.HTTPClient)
	}

	switch fullCmd {
	case scriptCmd.FullCommand():
		script(ctx, *scriptArgs.file)

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

func script(ctx *mansion.Context, scriptPath string) {
	ctx.Must(doScript(scriptPath))
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

