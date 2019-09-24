package installer

import (
	"io"
	"path/filepath"
	"time"

	"github.com/itchio/boar"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

func GetInstallerInfo(consumer *state.Consumer, file eos.File) (*InstallerInfo, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	target := stat.Name()
	ext := filepath.Ext(target)
	name := filepath.Base(target)

	consumer.Infof("↝ For source (%s)", name)

	var installerType = InstallerTypeUnknown

	if extType, ok := installerForExt[ext]; ok {
		consumer.Infof("✓ Using file extension registry (%s) => (%s)", ext, extType)
		installerType = extType
	} else {
		consumer.Warnf("  No mapping for file extension (%s)", ext)
	}

	if installerType == InstallerTypeArchive {
		// some archive types are better sniffed by 7-zip and/or butler's own
		// decompression engines, so if configurator returns naked, we try
		// to open as an archive.
		beforeArchiveProbe := time.Now()
		consumer.Infof("  Probing with boar (because installer type is archive)...")

		var entries []*savior.Entry
		archiveInfo, err := boar.Probe(&boar.ProbeParams{
			File:     file,
			Consumer: consumer,
			OnEntries: func(es []*savior.Entry) {
				entries = es
			},
		})
		consumer.Debugf("  (archive probe took %s)", time.Since(beforeArchiveProbe))
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if archiveInfo == nil {
			consumer.Infof("✗ Source is not a supported archive format")
		} else {
			consumer.Infof("✓ Source is a supported archive format (%s)", archiveInfo.Format)
			consumer.Infof("  Features: %s", archiveInfo.Features)
			return &InstallerInfo{
				Type:        InstallerTypeArchive,
				ArchiveInfo: archiveInfo,
				Entries:     entries,
			}, nil
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &InstallerInfo{
		Type: installerType,
	}, nil
}

func getInstallerTypeForCandidate(consumer *state.Consumer, candidate *dash.Candidate, file eos.File) (InstallerType, error) {
	switch candidate.Flavor {

	case dash.FlavorNativeWindows:
		if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
			consumer.Infof("  → Found Windows installer of type %s", candidate.WindowsInfo.InstallerType)
			consumer.Warnf("  But support for Windows installers has been removed, so, we'll just copy them in place.")
		} else {
			consumer.Infof("  → Native Windows executable, sure hope it's not an installer")
		}

		return InstallerTypeNaked, nil

	case dash.FlavorNativeMacos:
		consumer.Infof("  → Native macOS executable")
		return InstallerTypeNaked, nil

	case dash.FlavorNativeLinux:
		consumer.Infof("  → Native linux executable")
		return InstallerTypeNaked, nil

	case dash.FlavorScript:
		consumer.Infof("  → Script")
		if candidate.ScriptInfo != nil && candidate.ScriptInfo.Interpreter != "" {
			consumer.Infof("    with interpreter %s", candidate.ScriptInfo.Interpreter)
		}
		return InstallerTypeNaked, nil

	case dash.FlavorScriptWindows:
		consumer.Infof("  → Windows script")
		return InstallerTypeNaked, nil
	}

	return InstallerTypeUnknown, nil
}
