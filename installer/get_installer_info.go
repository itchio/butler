package installer

import (
	"io"
	"path/filepath"
	"time"

	"github.com/itchio/boar"
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
		archiveInfo, err := boar.Probe(boar.ProbeParams{
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
