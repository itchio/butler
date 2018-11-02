package damage

import (
	"github.com/itchio/damage/hdiutil"
	"github.com/pkg/errors"
)

type MountResult struct {
	SystemEntities []SystemEntity `plist:"system-entities"`
}

type SystemEntity struct {
	MountPoint           string `plist:"mount-point"`
	PotentiallyMountable bool   `plist:"potentially-mountable"`
	UnmappedContentHint  string `plist:"unmapped-content-hint"`
	VolumeKind           string `plist:"volume-kind"`
	ContentHint          string `plist:"content-hint"`
	DevEntry             string `plist:"dev-entry"`
}

// Mount a dmg file into a directory
func Mount(host hdiutil.Host, dmgpath string, dir string) (*MountResult, error) {
	var res MountResult
	err := host.Command("attach").WithArgs(
		"-plist",             // output format
		"-nobrowse",          // don't show in Finder
		"-noverify",          // we already verify image checksums when downloading
		"-noautofsck",        // nuh-huh
		"-noautoopen",        // please don't
		"-mount", "required", // if we can't mount why bother?
		"-mountpoint", dir,
		"-readonly", // we won't ever write to it
		"-noidme",   // some kind of scripting, disable
		dmgpath,
	).WithInput("Y").RunAndDecode(&res)

	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &res, err
}

func Unmount(host hdiutil.Host, dir string) error {
	err := host.Command("detach").WithArgs(dir).Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
