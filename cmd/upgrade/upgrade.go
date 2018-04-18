package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/itchio/boar"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	head   *bool
	stable *bool
}{}

func Register(ctx *mansion.Context) {
	// aliases include common misspellings
	cmd := ctx.App.Command("upgrade", "Upgrades butler to the latest version").Alias("ugprade").Alias("update")
	ctx.Register(cmd, do)

	args.head = cmd.Flag("head", "Force bleeding-edge version").Bool()
	args.stable = cmd.Flag("stable", "Force stable version").Bool()
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.head, *args.stable))
}

func Do(ctx *mansion.Context, head bool, stable bool) error {
	variant := ctx.CurrentVariant()
	if head {
		variant = mansion.VersionVariantHead
	} else if stable {
		variant = mansion.VersionVariantStable
	}

	comm.Opf("Looking for %s upgrades...", variant)

	vinfo, err := ctx.QueryLatestVersion(variant)
	if err != nil {
		return fmt.Errorf("Version check failed: %s", err.Error())
	}

	if vinfo.Latest.Equal(vinfo.Current) {
		comm.Statf("Your butler is up-to-date. Have a nice day!")
		return nil
	}

	comm.Statf("Current version: %s", vinfo.Current)
	comm.Statf("Latest version : %s", vinfo.Latest)

	if !comm.YesNo("Do you want to upgrade now?") {
		comm.Logf("Okay, not upgrading. Bye!")
		return nil
	}

	return applyUpgrade(ctx, vinfo)
}

func applyUpgrade(ctx *mansion.Context, vinfo *mansion.VersionCheckResult) error {
	before := vinfo.Current
	after := vinfo.Latest

	updateDir, err := ioutil.TempDir("", "butler-self-upgrade")
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.RemoveAll(updateDir)

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execDir := filepath.Dir(execPath)

	archiveURL := fmt.Sprintf("%s/%s/.zip", ctx.UpdateBaseURL(after.Variant), after.Name)
	comm.Opf("%s", archiveURL)

	extractRes, err := boar.SimpleExtract(&boar.SimpleExtractParams{
		ArchivePath:       archiveURL,
		DestinationFolder: updateDir,
		Consumer:          comm.NewStateConsumer(),
	})
	if err != nil {
		return err
	}

	var oldPaths []string

	for _, entry := range extractRes.Entries {
		srcPath := filepath.Join(updateDir, entry.CanonicalPath)
		dstPath := filepath.Join(execDir, entry.CanonicalPath)
		oldPath := dstPath + ".old"

		oldPaths = append(oldPaths, oldPath)
		os.Rename(dstPath, oldPath)
		os.Rename(srcPath, dstPath)
	}

	for _, oldPath := range oldPaths {
		err = os.Remove(oldPath)
	}

	comm.Statf("Upgraded butler from %s to %s. Have a nice day!", before, after)
	return nil
}
