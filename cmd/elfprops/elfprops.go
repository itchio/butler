package elfprops

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"sort"
	"strings"

	"github.com/itchio/httpkit/timeout"
	"github.com/pkg/errors"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
)

var args = struct {
	path          string
	trace         bool
	analyzeDistro string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	cmd.Arg("path", "The ELF binary to analyze").Required().StringVar(&args.path)
	cmd.Flag("trace", "Also perform a dependency trace (will probably only work on Linux)").BoolVar(&args.trace)
	cmd.Flag("analyze-distro", "Also print a list of non-default ubuntu packages for a distro").Hidden().StringVar(&args.analyzeDistro)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()

	f, err := eos.Open(args.path, option.WithConsumer(consumer))
	ctx.Must(err)
	defer f.Close()

	info, err := Do(f, comm.NewStateConsumer())
	ctx.Must(err)

	comm.ResultOrPrint(info, func() {
		js, err := json.MarshalIndent(info, "", "  ")
		if err == nil {
			comm.Logf("%s", string(js))
		}
	})

	if args.trace {
		root, err := elefant.Trace(info, args.path)
		ctx.Must(err)

		comm.Logf("%s", root)

		if args.analyzeDistro != "" {
			codeword := args.analyzeDistro
			consumer.Infof("Analyzing for distro %s (%s)", codeword, info.Arch)

			debarch := ""
			switch info.Arch {
			case elefant.Arch386:
				debarch = "i386"
			case elefant.ArchAmd64:
				debarch = "amd64"
			default:
				ctx.Must(errors.Errorf("Don't know equivalent debian arch for (%s)", info.Arch))
			}

			nameMap := make(map[string]bool)
			for _, c := range root.Children {
				if c.FullPath != "" {
					nameMap[c.FullPath] = true
				}
			}

			var sortedNames []string
			for k := range nameMap {
				sortedNames = append(sortedNames, k)
			}
			sort.Strings(sortedNames)

			log.Printf("Looking for: %s", strings.Join(sortedNames, ", "))

			manifestURL, err := getManifestURL(codeword, debarch)
			ctx.Must(err)

			consumer.Infof("Downloading manifest (%s)...", manifestURL)
			client := timeout.NewDefaultClient()
			res, err := client.Get(manifestURL)
			ctx.Must(err)
			defer res.Body.Close()

			if res.StatusCode != 200 {
				err = errors.Errorf("Got HTTP %d for %s", res.StatusCode, manifestURL)
				ctx.Must(err)
			}

			manifestBytes, err := ioutil.ReadAll(res.Body)
			ctx.Must(err)

			basePackageMap := make(map[string]bool)
			for _, line := range strings.Split(string(manifestBytes), "\n") {
				packageName := strings.Split(line, " ")[0]
				archlessPackageName := strings.Split(packageName, ":")[0]
				basePackageMap[archlessPackageName] = true
			}
			log.Printf("Recorded %d base packages", len(basePackageMap))

			type PackageResponse struct {
				Packages []string `json:"packages"`
			}

			requiredPackages := make(map[string]bool)
			processLib := func(libName string) {
				values := make(url.Values)
				values.Add("file", libName)
				searchURL := fmt.Sprintf("https://broth.itch.zone/debsearch/by-file-path/ubuntu/%s/%s?%s", codeword, debarch, values.Encode())
				log.Printf("Querying broth (%s)...", searchURL)

				res, err := client.Get(searchURL)
				ctx.Must(err)

				if res.StatusCode != 200 {
					log.Printf("Ignoring, got HTTP %d for %s", res.StatusCode, searchURL)
					return
				}

				defer res.Body.Close()
				resBytes, err := ioutil.ReadAll(res.Body)
				ctx.Must(err)

				var pr PackageResponse
				ctx.Must(json.Unmarshal(resBytes, &pr))

				for _, p := range pr.Packages {
					if strings.HasSuffix(p, "-dbg") {
						continue
					}
					if strings.HasSuffix(p, "-cross") {
						continue
					}

					requiredPackages[p] = true
					break
				}
			}

			for _, libName := range sortedNames {
				processLib(libName)
				consumer.Infof("%d packages found so far ...", len(requiredPackages))
			}

			printSortedKeys := func(m map[string]bool) {
				var keys []string
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					consumer.Infof(" - %s", k)
				}
			}

			consumer.Statf("%d packages found in total:", len(requiredPackages))
			printSortedKeys(requiredPackages)

			for pkg := range requiredPackages {
				if _, ok := basePackageMap[pkg]; ok {
					delete(requiredPackages, pkg)
				}
			}

			consumer.Statf("%d non-base packages remaining:", len(requiredPackages))
			printSortedKeys(requiredPackages)
		}
	}
}

func getManifestURL(codeword string, debarch string) (string, error) {
	ubuntuManifestURL := func(v string) string {
		return fmt.Sprintf("http://releases.ubuntu.com/%s/ubuntu-%s-desktop-%s.manifest", v, v, debarch)
	}

	switch codeword {
	case "xenial":
		return ubuntuManifestURL("16.04.4"), nil
	default:
		return "", errors.Errorf("Unknown codeword: %s", codeword)
	}
}

func Do(f eos.File, consumer *state.Consumer) (*elefant.ElfInfo, error) {
	return elefant.Probe(f, elefant.ProbeParams{
		Consumer: consumer,
	})
}
