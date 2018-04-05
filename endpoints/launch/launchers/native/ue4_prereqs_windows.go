// +build windows

package native

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/itchio/pelican"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/runner/winutil"
	"github.com/pkg/errors"
)

type UE4Marker struct {
	SHA256 string
}

func getMarkerPath(params *launch.LauncherParams) string {
	return filepath.Join(params.InstallFolder, ".itch", "ue4-prereqs-marker.txt")
}

func readUE4Marker(params *launch.LauncherParams) (*UE4Marker, error) {
	markerPath := getMarkerPath(params)
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

func writeUE4Marker(params *launch.LauncherParams, marker *UE4Marker) error {
	payload, err := json.Marshal(marker)
	if err != nil {
		return errors.WithStack(err)
	}

	markerPath := getMarkerPath(params)
	err = os.MkdirAll(filepath.Dir(markerPath), 0755)
	if err != nil {
		return errors.WithStack(err)
	}

	err = ioutil.WriteFile(markerPath, payload, 0644)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
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

	installContainer, err := params.GetInstallContainer()
	if err != nil {
		return errors.WithStack(err)
	}

	var prereqCandidate *configurator.Candidate

	for _, fe := range installContainer.Files {
		consumer.Infof("Reviewing (%s)", path.Base(fe.Path))
		if path.Base(fe.Path) == needle {
			prereqCandidate, err = params.SniffFile(fe)
			if err != nil {
				return errors.WithStack(err)
			}
			break
		}
	}

	if prereqCandidate == nil {
		return nil
	}

	consumer.Infof("Found UE4 prereq candidate:\n %s", prereqCandidate)

	prereqCandidatePath := filepath.Join(params.InstallFolder, prereqCandidate.Path)
	f, err := os.Open(prereqCandidatePath)
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
	sha256String := fmt.Sprintf("%x", sha256Bytes)

	if marker == nil {
		consumer.Infof("SHA256 of UE4 prereq: %s", sha256String)
		consumer.Infof("No marker")
	} else {
		consumer.Infof("SHA256 of UE4 prereq: %s", sha256String)
		consumer.Infof("SHA256 of UE4 marker: %s", marker.SHA256)
		if sha256String == marker.SHA256 {
			consumer.Infof("UE4 prereqs already installed!")
			if params.ForcePrereqs {
				consumer.Infof("Prereqs forced, installing anyway...")
			} else {
				return nil
			}
		}
	}

	err = winutil.VerifyTrust(prereqCandidatePath)
	if err != nil {
		return errors.WithMessage(err, "while verifying UE4 prereqs signature")
	}

	consumer.Infof("Authenticode signature verified.")
	args := []string{
		"elevate",
		"--",
		prereqCandidatePath,
		"/quiet",
		"/norestart",
	}

	consumer.Infof("Attempting elevated UE4 prereqs install")
	installRes, err := installer.RunSelf(&installer.RunSelfParams{
		Consumer: consumer,
		Args:     args,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if installRes.ExitCode != 0 {
		if installRes.ExitCode == elevate.ExitCodeAccessDenied {
			msg := "User or system did not grant elevation privileges"
			consumer.Errorf(msg)
			return errors.WithStack(butlerd.CodeOperationAborted)
		}

		consumer.Errorf("UE4 prereq install failed (code %d, 0x%x), we're out of options", installRes.ExitCode, installRes.ExitCode)
		return errors.New("UE4 prereq installation failed")
	}

	err = writeUE4Marker(params, &UE4Marker{
		SHA256: fmt.Sprintf("%x", sha256Bytes),
	})
	if err != nil {
		return errors.WithMessage(err, "while writing ue4 marker")
	}

	return nil
}
