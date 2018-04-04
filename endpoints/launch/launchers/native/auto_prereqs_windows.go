// +build windows

package native

import (
	"path/filepath"
	"strings"

	"github.com/itchio/butler/redist"

	"github.com/itchio/butler/cmd/prereqs"

	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/pelican"
	"github.com/itchio/wharf/eos"
	"github.com/pkg/errors"
)

func handleAutoPrereqs(params *launch.LauncherParams, pc *prereqs.PrereqsContext) ([]string, error) {
	consumer := params.RequestContext.Consumer

	candidate := params.Candidate
	if candidate == nil {
		// can't do auto prereqs if we have no candidate!
		return nil, nil
	}

	if candidate.Flavor != configurator.FlavorNativeWindows {
		// can only do auto prereqs on native executable
		return nil, nil
	}

	candidateArch := candidate.Arch

	consumer.Opf("Determining dependencies for all (%s) executables in (%s)", candidateArch, params.InstallFolder)

	verdict, err := params.GetUnfilteredVerdict()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	importsMap := make(map[string]bool)

	handleCandidate := func(c *configurator.Candidate) error {
		if c.Flavor != configurator.FlavorNativeWindows {
			return nil
		}

		if c.Arch != candidateArch {
			return nil
		}

		cPath := filepath.Join(params.InstallFolder, c.Path)

		f, err := eos.Open(cPath)
		if err != nil {
			consumer.Warnf("For auto prereqs: could not open (%s): %s", cPath, err.Error())
			return nil
		}
		defer f.Close()

		peInfo, err := pelican.Probe(f, &pelican.ProbeParams{
			Consumer: consumer,
		})
		if err != nil {
			return err
		}

		for _, imp := range peInfo.Imports {
			importsMap[strings.ToLower(imp)] = true
		}
		return nil
	}

	for _, c := range verdict.Candidates {
		err = handleCandidate(c)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	consumer.Opf("Mapping dependencies to prereqs...")

	regist, err := pc.GetRegistry()
	if err != nil {
		return nil, err
	}

	dllToRedistMap := make(map[string]string)
	for redistName, redist := range regist.Entries {
		if redist.Windows != nil {
			for _, dll := range redist.Windows.DLLs {
				dllToRedistMap[strings.ToLower(dll)] = redistName
			}
		}
	}

	wantedMap := make(map[string]bool)

	for imp, _ := range importsMap {
		var bestEntry *redist.RedistEntry
		var bestEntryName string

		if entryName, ok := dllToRedistMap[imp]; ok {
			entry := regist.Entries[entryName]
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
	for entryName, _ := range wantedMap {
		wanted = append(wanted, entryName)
	}

	return wanted, nil
}
