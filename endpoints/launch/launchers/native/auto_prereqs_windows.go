// +build windows

package native

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/itchio/butler/redist"

	"github.com/itchio/butler/cmd/prereqs"

	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/dash"
	"github.com/itchio/pelican"
	"github.com/pkg/errors"
)

func handleAutoPrereqs(params launch.LauncherParams, pc *prereqs.PrereqsContext) ([]string, error) {
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

		peInfo, err := pelican.Probe(f, &pelican.ProbeParams{
			Consumer: consumer,
		})
		if err != nil {
			consumer.Warnf("For auto prereqs: could not probe (%s): %+v", cPath, err)
			return nil
		}

		for _, imp := range peInfo.Imports {
			importsMap[strings.ToLower(imp)] = true
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

	consumer.Opf("Mapping dependencies to prereqs...")

	registry, err := pc.GetRegistry()
	if err != nil {
		return nil, err
	}

	dllToRedistMap := make(map[string]string)

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
				dllToRedistMap[strings.ToLower(dll)] = redistName
			}
		}
	}

	wantedMap := make(map[string]bool)

	for imp := range importsMap {
		var bestEntry *redist.RedistEntry
		var bestEntryName string

		if entryName, ok := dllToRedistMap[imp]; ok {
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
