package native

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/itchio/butler/cmd/prereqs"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/butler/redist"
	"github.com/itchio/ox"

	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/pelican"

	"github.com/pkg/errors"
)

func handleAutoPrereqs(params launch.LauncherParams, h prereqs.Handler) ([]string, error) {
	switch params.Host.Runtime.Platform {
	case ox.PlatformWindows:
		return handleAutoPrereqsWindows(params, h)
	default:
		// no auto prereqs on non-windows platforms for now
		return nil, nil
	}
}

func handleAutoPrereqsWindows(params launch.LauncherParams, h prereqs.Handler) ([]string, error) {
	consumer := params.RequestContext.Consumer

	candidate := params.Candidate
	if candidate == nil {
		// can't do auto prereqs if we have no candidate!
		return nil, nil
	}

	if candidate.Flavor != dash.FlavorNativeWindows {
		// can only do auto prereqs on native executable
		return nil, nil
	}

	candidateArch := candidate.Arch

	consumer.Opf("Determining dependencies for all (%s) executables in (%s)", candidateArch, params.InstallFolder)

	installContainer, err := params.GetInstallContainer()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	importsMap := make(map[string]bool)

	handleCandidate := func(c *dash.Candidate) error {
		if c.Flavor != dash.FlavorNativeWindows {
			return nil
		}

		if c.Arch != candidateArch {
			return nil
		}

		cPath := filepath.Join(params.InstallFolder, c.Path)

		f, err := os.Open(cPath)
		if err != nil {
			consumer.Warnf("For auto prereqs: could not open (%s): %v", cPath, err)
			return nil
		}
		defer f.Close()

		var peLines []string
		memConsumer := &state.Consumer{
			OnMessage: func(lvl string, msg string) {
				peLines = append(peLines, fmt.Sprintf("[%s] %s", lvl, msg))
			},
		}

		peInfo, err := pelican.Probe(f, pelican.ProbeParams{
			Consumer: memConsumer,
		})
		if err != nil {
			consumer.Warnf("For auto prereqs: could not probe (%s): %+v", cPath, err)
			consumer.Warnf("Full pelican log:\n%s", strings.Join(peLines, "\n"))
			return nil
		}

		cFolder := filepath.Dir(cPath)
		vendorDLLs := make(map[string]bool)
		{
			fileInfos, err := ioutil.ReadDir(cFolder)
			if err != nil {
				consumer.Warnf("For auto prereqs: could not list folder (%s): %v", cFolder, err)
				return nil
			}

			for _, fi := range fileInfos {
				lowerName := strings.ToLower(fi.Name())
				if strings.HasSuffix(lowerName, ".dll") {
					vendorDLLs[lowerName] = true
				}
			}
		}

		for _, imp := range peInfo.Imports {
			lowerDLL := strings.ToLower(imp)
			isVendored := vendorDLLs[lowerDLL]
			if isVendored {
				consumer.Infof("Found vendored DLL, skipping prereqs for (%s)", lowerDLL)
			} else if desc, isBuiltin := knownBuiltinDLLs[lowerDLL]; isBuiltin {
				consumer.Infof("Found built-in DLL, skipping prereqs for (%s) (%s)", lowerDLL, desc)
			} else {
				importsMap[lowerDLL] = true
			}
		}
		return nil
	}

	for _, fe := range installContainer.Files {
		if strings.HasSuffix(strings.ToLower(fe.Path), ".exe") {
			c, err := params.SniffFile(fe)
			if err != nil {
				return nil, errors.WithStack(err)
			}

			err = handleCandidate(c)
			if err != nil {
				return nil, errors.WithStack(err)
			}
		}
	}

	if len(importsMap) == 0 {
		consumer.Opf("No DLL imports to map to prereqs!")
		return nil, nil
	}

	consumer.Opf("Mapping (%d) dependencies to prereqs:", len(importsMap))
	for imp := range importsMap {
		consumer.Infof(" - (%s)", imp)
	}

	registry, err := h.GetRegistry()
	if err != nil {
		return nil, err
	}

	dllToRedistMap := make(map[string][]string)

	var redistNames []string
	// map iteration is random in go (they mean it)
	// so we have to sort it here. cf. https://github.com/itchio/itch/issues/1754
	for redistName := range registry.Entries {
		redistNames = append(redistNames, redistName)
	}
	sort.Strings(redistNames)

	for _, redistName := range redistNames {
		redist := registry.Entries[redistName]
		if redist.Windows != nil {
			for _, dll := range redist.Windows.DLLs {
				k := strings.ToLower(dll)
				dllToRedistMap[k] = append(dllToRedistMap[k], redistName)
			}
		}
	}

	wantedMap := make(map[string]bool)

	for imp := range importsMap {
		var bestEntry *redist.RedistEntry
		var bestEntryName string

		if entryNames, ok := dllToRedistMap[imp]; ok {
			for _, entryName := range entryNames {
				entry := registry.Entries[entryName]
				if bestEntry == nil {
					bestEntry = entry
					bestEntryName = entryName
				} else {
					if entry.Arch == string(candidateArch) {
						// prefer matching arch, if we have that luxury
						bestEntry = entry
						bestEntryName = entryName
					}
				}
			}
		}

		if bestEntryName != "" {
			consumer.Statf("%s ships with prereq %s", imp, bestEntryName)
			wantedMap[bestEntryName] = true
		}
	}

	var wanted []string
	for entryName := range wantedMap {
		wanted = append(wanted, entryName)
	}

	return wanted, nil
}
