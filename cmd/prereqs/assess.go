package prereqs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/itchio/butler/redist"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

type PrereqAssessment struct {
	Done []string
	Todo []string
}

func (pc *PrereqsContext) AssessPrereqs(names []string) (*PrereqAssessment, error) {
	pa := &PrereqAssessment{}

	for _, name := range names {
		entry, err := pc.GetEntry(name)
		if entry == nil {
			pc.Consumer.Warnf("Prereq (%s) not found in registry, skipping...", name)
			continue
		}

		alreadyGood := false

		switch pc.Runtime.Platform {
		case ox.PlatformWindows:
			alreadyGood, err = pc.AssessWindowsPrereq(name, entry)
			if err != nil {
				return nil, errors.Wrap(err, "assessing windows prereq")
			}
		case ox.PlatformLinux:
			alreadyGood, err = pc.AssessLinuxPrereq(name, entry)
			if err != nil {
				return nil, errors.Wrap(err, "assessing linux prereq")
			}
		}

		if alreadyGood {
			// then it's already installed, cool!
			pa.Done = append(pa.Done, name)
			continue
		}

		pa.Todo = append(pa.Todo, name)
	}

	for _, name := range pa.Done {
		err := pc.MarkInstalled(name)
		if err != nil {
			return nil, errors.Wrapf(err, "marking %s as installed", name)
		}
		continue
	}

	return pa, nil
}

func (pc *PrereqsContext) MarkerPath(name string) string {
	return filepath.Join(pc.PrereqsDir, name, ".installed")
}

func (pc *PrereqsContext) HasInstallMarker(name string) bool {
	path := pc.MarkerPath(name)
	_, err := os.Stat(path)
	return err == nil
}

func (pc *PrereqsContext) MarkInstalled(name string) error {
	if pc.HasInstallMarker(name) {
		// don't mark again
		return nil
	}

	contents := fmt.Sprintf("Installed on %s", time.Now())
	path := pc.MarkerPath(name)
	err := os.MkdirAll(filepath.Dir(path), os.FileMode(0o755))
	if err != nil {
		return errors.Wrap(err, "creating marker dir")
	}

	err = ioutil.WriteFile(path, []byte(contents), os.FileMode(0o644))
	if err != nil {
		return errors.Wrap(err, "writing marker file")
	}

	return nil
}

func (pc *PrereqsContext) AssessWindowsPrereq(name string, entry *redist.RedistEntry) (bool, error) {
	block := entry.Windows

	for _, registryKey := range block.RegistryKeys {
		if RegistryKeyExists(pc.Consumer, registryKey) {
			pc.Consumer.Debugf("Found registry key (%s)", registryKey)
			return true, nil
		}
	}

	return false, nil
}

func (pc *PrereqsContext) AssessLinuxPrereq(name string, entry *redist.RedistEntry) (bool, error) {
	block := entry.Linux

	switch block.Type {
	case redist.LinuxRedistTypeHosted:
		// cool!
	default:
		return false, fmt.Errorf("Don't know how to assess linux prereq of type (%s)", block.Type)
	}

	for _, sc := range block.SanityChecks {
		err := pc.RunSanityCheck(name, entry, sc)
		if err != nil {
			return false, nil
		}
	}

	return true, nil
}

func (pc *PrereqsContext) RunSanityCheck(name string, entry *redist.RedistEntry, sc *redist.LinuxSanityCheck) error {
	cmd := exec.Command(sc.Command, sc.Args...)
	cmd.Dir = pc.GetEntryDir(name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		pc.Consumer.Debugf("Sanity check failed:%s\n%s", err.Error(), string(output))
		return errors.Wrapf(err, "performing sanity check for %s", name)
	}

	return nil
}
