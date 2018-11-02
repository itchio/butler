package dmg

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/butler/installer/dmg/dmgextract"

	"github.com/itchio/butler/installer"
	"github.com/pkg/errors"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	var res = installer.InstallResult{
		Files: []string{},
	}

	f := params.File

	localFile, err := installer.AsLocalFile(f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Infof("DMG installer tuning in")

	mountFolder := filepath.Join(params.StageFolderPath, "dmg-mountpoint")
	consumer.Infof("Will use (%s) as a mount point", mountFolder)
	err = os.MkdirAll(mountFolder, 0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumer.Infof("Invoking dmgextract")
	exRes, err := dmgextract.New(
		localFile.Name(),
		dmgextract.ExtractSLA,
		dmgextract.WithConsumer(consumer),
		dmgextract.WithMountFolder(mountFolder),
	).ExtractTo(params.InstallFolderPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	container := exRes.Container

	for _, file := range container.Files {
		res.Files = append(res.Files, file.Path)
	}
	for _, symlink := range container.Symlinks {
		res.Files = append(res.Files, symlink.Path)
	}
	for _, dir := range container.Dirs {
		res.Files = append(res.Files, dir.Path)
	}

	consumer.Opf("Busting ghosts...")
	err = bfs.BustGhosts(&bfs.BustGhostsParams{
		Folder:   params.InstallFolderPath,
		NewFiles: res.Files,
		Receipt:  params.ReceiptIn,

		Consumer: params.Consumer,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	slaPath := filepath.Join(params.InstallFolderPath, ".itch", "sla.txt")
	if exRes.SLA == nil {
		_, err := os.Stat(slaPath)
		if err != nil {
			consumer.Opf("Wiping pre-existing SLA at (%s)", slaPath)
			os.Remove(slaPath)
		}
	} else {
		consumer.Opf("Writing SLA to (%s)", slaPath)
		err = os.MkdirAll(filepath.Dir(slaPath), 0755)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		err = ioutil.WriteFile(slaPath, []byte(exRes.SLA.Text), 0644)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return &res, nil
}
