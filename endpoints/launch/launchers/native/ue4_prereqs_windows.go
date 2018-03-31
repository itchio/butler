// +build windows

package native

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/pelican"
	"github.com/itchio/wharf/eos"

	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/pkg/errors"
)

type UE4Marker struct {
	SHA256 string
}

func readUE4Marker(params *launch.LauncherParams) (*UE4Marker, error) {
	markerPath := filepath.Join(params.InstallFolder, ".itch", "ue4-prereqs-done.txt")
	markerBytes, err := ioutil.ReadFile(markerPath)
	if err != nil {
		if os.IsNotExist(err) {
			// ok
			return nil, nil
		}
		return nil, errors.WithMessage(err, "while checking for ue4 prereq marker")
	}

	marker := &UE4Marker{}
	err = json.Unmarshal(markerBytes, marker)
	if err != nil {
		return nil, errors.WithMessage(err, "while decoding ue4 prereq marker")
	}

	return marker, nil
}

func handleUE4Prereqs(params *launch.LauncherParams) error {
	marker, err := readUE4Marker(params)
	if err != nil {
		return errors.WithStack(err)
	}

	consumer := params.RequestContext.Consumer

	if params.PeInfo != nil {
		prettyPeInfoBytes, err := json.MarshalIndent(params.PeInfo, "", "  ")
		if err == nil {
			consumer.Infof("PE info: %s", string(prettyPeInfoBytes))
		}
	}

	if params.Candidate == nil {
		return nil
	}

	var needle string
	switch params.Candidate.Arch {
	case configurator.Arch386:
		needle = "UE4PrereqSetup_x86.exe"
	case configurator.ArchAmd64:
		needle = "UE4PrereqSetup_x64.exe"
	}

	if needle == "" {
		return nil
	}

	res, err := configurator.Configure(params.InstallFolder, false)
	if err != nil {
		return errors.WithStack(err)
	}

	var prereqCandidate *configurator.Candidate

	for _, c := range res.Candidates {
		base := filepath.Base(c.Path)
		if base == needle {
			prereqCandidate = c
			break
		}
	}

	if prereqCandidate == nil {
		return nil
	}

	consumer.Infof("Found UE4 prereq candidate:\n %s", prereqCandidate)

	prereqCandidatePath := filepath.Join(params.InstallFolder, prereqCandidate.Path)
	f, err := eos.Open(prereqCandidatePath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	peInfo, err := pelican.Probe(f, &pelican.ProbeParams{
		Consumer: consumer,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	prettyPeInfoBytes, err := json.MarshalIndent(peInfo, "", "  ")
	if err == nil {
		consumer.Infof("Prereq PE info: %s", string(prettyPeInfoBytes))
	}

	var expectedFileDescription string
	switch params.Candidate.Arch {
	case configurator.Arch386:
		expectedFileDescription = "UE4 Prerequisites (x86)"
	case configurator.ArchAmd64:
		expectedFileDescription = "UE4 Prerequisites (x64)"
	}

	fileDescription := peInfo.VersionProperties["FileDescription"]
	if fileDescription != expectedFileDescription {
		consumer.Warnf("Ignoring (%s), was expecting (%s) but got (%s)", prereqCandidate.Path, expectedFileDescription, fileDescription)
		return nil
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return errors.WithMessage(err, "seeking to start of UE4 prereq")
	}

	hash := sha256.New()
	_, err = io.Copy(hash, f)
	if err != nil {
		return errors.WithMessage(err, "computing sha256 of UE4 prereq")
	}

	sha256Bytes := hash.Sum(nil)

	if marker == nil {
		consumer.Infof("SHA256 of UE4 prereq: %x", sha256Bytes)
		consumer.Infof("No marker")
	} else {
		consumer.Infof("SHA256 of UE4 prereq: %x", sha256Bytes)
		consumer.Infof("SHA256 of UE4 marker: %s", marker.SHA256)
	}

	return nil
}
