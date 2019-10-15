package launch

import (
	"bytes"
	"crypto/sha256"
	"io/ioutil"
	"path/filepath"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

// ensureLicenseAcceptance checks whether we need the user to accept
// a license before continuing.
func ensureLicenseAcceptance(rc *butlerd.RequestContext, installFolder string) error {
	consumer := rc.Consumer

	license := getLicense(installFolder)
	if license == "" {
		consumer.Debugf("No license agreement, continuing")
		return nil
	}

	hashed := hashLicense(license)
	marker := getLicenseMarker(installFolder)
	if bytes.Equal(hashed, marker) {
		consumer.Infof("License agreement already accepted")
		return nil
	}

	if marker == nil {
		consumer.Infof("Found license, never accepted before")
	} else {
		consumer.Infof("Found license, a different one was accepted before")
	}

	res, err := messages.AcceptLicense.Call(rc, butlerd.AcceptLicenseParams{
		Text: license,
	})
	if err != nil {
		return err
	}

	if !res.Accept {
		consumer.Errorf("License rejected, cancelling launch")
		return butlerd.CodeOperationCancelled
	}

	err = writeLicenseMarker(installFolder, hashed)
	if err != nil {
		consumer.Warnf("Could not write license marker: %+v", err)
	}
	return nil
}

func licensePath(installFolder string) string {
	return filepath.Join(installFolder, ".itch", "sla.txt")
}

func licenseMarkerPath(installFolder string) string {
	return filepath.Join(installFolder, ".itch", "sla-accepted-hash.sha256")
}

func getLicense(installFolder string) string {
	payload, err := ioutil.ReadFile(licensePath(installFolder))
	if err != nil {
		return ""
	}

	return string(payload)
}

func hashLicense(license string) []byte {
	sum := sha256.Sum256([]byte(license))
	return sum[:]
}

func getLicenseMarker(installFolder string) []byte {
	payload, err := ioutil.ReadFile(licenseMarkerPath(installFolder))
	if err != nil {
		return nil
	}

	return payload
}

func writeLicenseMarker(installFolder string, hashed []byte) error {
	return ioutil.WriteFile(licenseMarkerPath(installFolder), hashed, 0o644)
}
