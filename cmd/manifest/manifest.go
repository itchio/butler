package manifest

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/itchio/butler/configurator"

	"github.com/mitchellh/mapstructure"

	"github.com/BurntSushi/toml"
	humanize "github.com/dustin/go-humanize"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var checkArgs = struct {
	dir *string
}{}

func Register(ctx *mansion.Context) {
	parentCmd := ctx.App.Command("manifest", "Manage itch.io app manifests")

	{
		cmd := parentCmd.Command("check", "Check the presence and validity of a manifest in a build")
		checkArgs.dir = cmd.Arg("dir", "Name of folder to check for a manifest").Required().String()
		ctx.Register(cmd, doCheck)
	}
}

func doCheck(ctx *mansion.Context) {
	ctx.Must(Check(comm.NewStateConsumer()))
}

func Check(consumer *state.Consumer) error {
	warn := func(msg string, args ...interface{}) {
		consumer.Infof("")
		consumer.Infof("================== Warning ==================")
		consumer.Infof(msg, args...)
		consumer.Infof("=============================================")
		consumer.Infof("")
	}

	hasDir := false
	dir := *checkArgs.dir

	var manifestPath string
	dirStats, err := os.Stat(dir)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if dirStats.IsDir() {
		comm.Opf("Validating build directory %s", dir)
		manifestPath = manifest.Path(dir)
		hasDir = true
	} else {
		comm.Opf("Validating manifest only")
		manifestPath = dir
	}

	if !hasDir {
		warn("In manifest-only validation mode. Pass a valid build directory to perform further checks.")
	}

	showHeuristics := func() error {
		consumer.Infof("")
		consumer.Infof("Heuristics will be used to launch your project.")
		if hasDir {
			consumer.Infof("")
			consumer.Infof("Current heuristic results:")
			verdict, err := configurator.Configure(dir, false)
			if err != nil {
				return errors.Wrap(err, 0)
			}
			consumer.Infof("")
			consumer.Infof("%s", verdict.Candidates)
		} else {
			warn("Pass a complete build folder to see launch heuristic results")
		}
		return nil
	}

	stats, err := os.Stat(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			consumer.Infof("No manifest found (expected it to be at %s)", manifestPath)
			err := showHeuristics()
			if err != nil {
				return errors.Wrap(err, 0)
			}
			return nil
		}
		return errors.Wrap(err, 0)
	}

	consumer.Opf("Validating %s manifest at (%s)", humanize.IBytes(uint64(stats.Size())), manifestPath)

	var intermediate map[string]interface{}
	_, err = toml.DecodeFile(manifestPath, &intermediate)
	if err != nil {
		consumer.Errorf("Parse error:")
		return errors.Wrap(err, 0)
	}

	jsonIntermediate, err := json.MarshalIndent(intermediate, "", "  ")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Debugf("Intermediate:\n%s", string(jsonIntermediate))

	appManifest := &buse.Manifest{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      appManifest,
		ErrorUnused: true,
	})
	if err != nil {
		consumer.Errorf("Internal error:")
		return errors.Wrap(err, 0)
	}

	err = decoder.Decode(intermediate)
	if err != nil {
		warnOnly := false
		if mse, ok := err.(*mapstructure.Error); ok {
			warnOnly = true
			for _, e := range mse.Errors {
				if strings.Contains(e, "has invalid keys") {
					// cool!
				} else {
					warnOnly = false
					break
				}
			}
		}

		if warnOnly {
			warn("%s", err.Error())
		} else {
			consumer.Errorf("Decoding error:")
			return errors.Wrap(err, 0)
		}
	}

	_, err = toml.DecodeFile(manifestPath, appManifest)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	jsonManifest, err := json.MarshalIndent(appManifest, "", "  ")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Debugf("Manifest:\n%s", string(jsonManifest))

	consumer.Infof("")
	if len(appManifest.Actions) > 0 {
		consumer.Statf("Validating %d actions...", len(appManifest.Actions))
		for _, action := range appManifest.Actions {
			consumer.Infof("")
			consumer.Infof("  * Action '%s' (%s)", action.Name, action.Path)
			if action.Platform != "" {
				switch action.Platform {
				case buse.ItchPlatformLinux:
					consumer.Infof("    Only for Linux")
				case buse.ItchPlatformOSX:
					consumer.Infof("    Only for macOS")
				case buse.ItchPlatformWindows:
					consumer.Infof("    Only for Windows")
				default:
					warn("Unknown platform specified: (%s)", action.Platform)
				}
			}
			if action.Scope != "" {
				consumer.Infof("    Requests API scope (%s)", action.Scope)
			}
			if action.Sandbox {
				consumer.Infof("    Sandbox opt-in")
			}
			if action.Console {
				consumer.Infof("    Console")
			}
			if len(action.Args) > 0 {
				consumer.Infof("    Passes arguments: %s", strings.Join(action.Args, " ::: "))
			}
		}
	} else {
		consumer.Statf("No actions found.")
		err := showHeuristics()
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	consumer.Infof("")
	if len(appManifest.Prereqs) > 0 {
		consumer.Statf("Validating %d prereqs...", len(appManifest.Prereqs))
		consumer.Infof("")

		regFile, err := eos.Open("https://dl.itch.ovh/itch-redists/info.json")
		if err != nil {
			return errors.Wrap(err, 0)
		}

		reg := &redist.RedistRegistry{}
		err = json.NewDecoder(regFile).Decode(reg)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for _, p := range appManifest.Prereqs {
			entry := reg.Entries[p.Name]
			if entry == nil {
				warn("Unknown prerequisite listed: %s", p.Name)
				continue
			}
			consumer.Infof("  * %s (%s)", entry.FullName, p.Name)
			var platforms []string
			if entry.Windows != nil {
				platforms = append(platforms, "Windows")
			}
			if entry.Linux != nil {
				platforms = append(platforms, "Linux")
			}
			if entry.OSX != nil {
				platforms = append(platforms, "macOS")
			}
			consumer.Infof("    Available on %s", strings.Join(platforms, ", "))
			consumer.Infof("    For architecture %s", entry.Arch)
			consumer.Infof("")
		}
	} else {
		consumer.Statf("No prereqs listed.")
		consumer.Infof("")
		consumer.Infof("If your application needs some libraries to pre-installed (.NET, Visual C++ Runtime, etc.),")
		consumer.Infof("you can list them in the manifest.")
		consumer.Infof("")
		consumer.Infof("Visit https://itch.io/docs/itch/integrating/manifest.html for more information.")
	}

	return nil
}
