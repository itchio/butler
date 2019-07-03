package installer

import (
	"io"
	"path/filepath"
	"time"

	"github.com/itchio/boar"
	"github.com/itchio/dash"
	"github.com/itchio/pelican"
	"github.com/itchio/savior"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/headway/state"
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

	if typ, ok := installerForExt[ext]; ok {
		if typ == InstallerTypeArchive {
			// let code flow, probe it as archive
		} else {
			consumer.Infof("✓ Using file extension registry (%s)", typ)
			return &InstallerInfo{
				Type: typ,
			}, nil
		}
	}

	// configurator is what we do first because it's generally fast:
	// it shouldn't read *much* of the remote file, and with htfs
	// caching, it's even faster. whereas 7-zip might read a *bunch*
	// of an .exe file before it gives up

	consumer.Infof("  Probing with dash...")

	beforeConfiguratorProbe := time.Now()
	candidate, err := dash.Sniff(file, target, stat.Size())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	consumer.Debugf("  (took %s)", time.Since(beforeConfiguratorProbe))

	var typePerConfigurator = InstallerTypeUnknown

	if candidate != nil {
		consumer.Infof("  Candidate: %s", candidate.String())
		typePerConfigurator, err = getInstallerTypeForCandidate(consumer, candidate, file)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	} else {
		consumer.Infof("  No results from configurator")
	}

	if typePerConfigurator == InstallerTypeUnknown || typePerConfigurator == InstallerTypeNaked || typePerConfigurator == InstallerTypeArchive {
		// some archive types are better sniffed by 7-zip and/or butler's own
		// decompression engines, so if configurator returns naked, we try
		// to open as an archive.
		beforeArchiveProbe := time.Now()
		consumer.Infof("  Probing as archive...")

		// seek to start first because configurator may have seeked itself
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		var entries []*savior.Entry
		archiveInfo, err := boar.Probe(&boar.ProbeParams{
			File:      file,
			Consumer:  consumer,
			Candidate: candidate,
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

	consumer.Infof("✓ Using configurator results")
	return &InstallerInfo{
		Type: typePerConfigurator,
	}, nil
}

func getInstallerTypeForCandidate(consumer *state.Consumer, candidate *dash.Candidate, file eos.File) (InstallerType, error) {
	switch candidate.Flavor {

	case dash.FlavorNativeWindows:
		if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
			typ := (InstallerType)(candidate.WindowsInfo.InstallerType)
			consumer.Infof("  → Windows installer of type %s", typ)
			return typ, nil
		}

		_, err := file.Seek(0, io.SeekStart)
		if err != nil {
			return InstallerTypeUnknown, errors.WithStack(err)
		}

		peInfo, err := pelican.Probe(file, &pelican.ProbeParams{
			Consumer: consumer,
		})
		if err != nil {
			return InstallerTypeUnknown, errors.WithStack(err)
		}

		if peInfo.AssemblyInfo != nil {
			switch peInfo.AssemblyInfo.RequestedExecutionLevel {
			case "highestAvailable", "requireAdministrator":
				consumer.Infof("  → Unsupported Windows installer (requested execution level %s)", peInfo.AssemblyInfo.RequestedExecutionLevel)
				return InstallerTypeUnsupported, nil
			}

			if peInfo.AssemblyInfo.Description == "IExpress extraction tool" {
				consumer.Infof("  → Self-extracting CAB created with IExpress")
				return InstallerTypeIExpress, nil
			}
		} else {
			stats, err := file.Stat()
			if err != nil {
				return InstallerTypeUnknown, errors.WithStack(err)
			}

			if HasSuspiciouslySetupLikeName(stats.Name()) {
				consumer.Infof("  → Unsupported Windows installer (no manifest, has name '%s')", stats.Name())
				return InstallerTypeUnsupported, nil
			}
		}

		consumer.Infof("  → Native windows executable, but not an installer")
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

func IsWindowsInstaller(typ InstallerType) bool {
	switch typ {
	case InstallerTypeMSI:
		return true
	case InstallerTypeNsis:
		return true
	case InstallerTypeInno:
		return true
	default:
		return false
	}
}
