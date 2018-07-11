package upgrade

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itchio/boar"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/pkg/errors"
)

var args = struct {
	head   bool
	stable bool
	force  bool
}{}

func Register(ctx *mansion.Context) {
	// aliases include common misspellings
	cmd := ctx.App.Command("upgrade", "Upgrades butler to the latest version").Alias("ugprade").Alias("update")
	ctx.Register(cmd, do)

	cmd.Flag("head", "Force bleeding-edge version").BoolVar(&args.head)
	cmd.Flag("stable", "Force stable version").BoolVar(&args.stable)
	cmd.Flag("force", "Force upgrade, even when using self-built butler").BoolVar(&args.force)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, args.head, args.stable, args.force))
}

func Do(ctx *mansion.Context, head bool, stable bool, force bool) error {
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

	if vinfo.Current.Name == "" && !force {
		comm.Warnf("Refusing to upgrade self-built butler without --force")
		return nil
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
	consumer := comm.NewStateConsumer()
	before := vinfo.Current
	after := vinfo.Latest

	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	execDir := filepath.Dir(execPath)

	updateDir := filepath.Join(execDir, ".butler-self-upgrade")
	err = os.MkdirAll(updateDir, 0755)
	if err != nil {
		return errors.WithStack(err)
	}
	defer os.RemoveAll(updateDir)

	archiveURL := fmt.Sprintf("%s/%s/.zip", ctx.UpdateBaseURL(after.Variant), after.Name)
	consumer.Opf("%s", archiveURL)

	comm.StartProgress()
	extractRes, err := boar.SimpleExtract(&boar.SimpleExtractParams{
		ArchivePath:       archiveURL,
		DestinationFolder: updateDir,
		Consumer:          consumer,
	})
	comm.EndProgress()
	if err != nil {
		return err
	}

	type Item struct {
		SourcePath string
		DestPath   string
		BackupPath string
	}

	var items []Item

	for _, entry := range extractRes.Entries {
		srcPath := filepath.Join(updateDir, entry.CanonicalPath)
		dstPath := filepath.Join(execDir, entry.CanonicalPath)
		oldPath := dstPath + ".old"

		items = append(items, Item{
			SourcePath: srcPath,
			DestPath:   dstPath,
			BackupPath: oldPath,
		})
	}

	backup := func() {
		for _, item := range items {
			os.Rename(item.DestPath, item.BackupPath)
		}
	}

	apply := func() error {
		for _, item := range items {
			err := os.Rename(item.SourcePath, item.DestPath)
			if err != nil {
				return err
			}
		}
		return nil
	}

	rollback := func() {
		for _, item := range items {
			os.Rename(item.BackupPath, item.DestPath)
		}
	}

	cleanup := func() {
		for _, item := range items {
			os.Remove(item.BackupPath)
		}
	}

	defer cleanup()

	backup()
	err = apply()
	if err != nil {
		rollback()
		return errors.Wrap(err, "Self-upgrade failed")
	}

	consumer.Statf("Upgraded butler from %s to %s. Have a nice day!", before, after)
	return nil
}
