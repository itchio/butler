package upgrade

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

var args = struct {
	head *bool
}{}

func Register(ctx *mansion.Context) {
	// aliases include common misspellings
	cmd := ctx.App.Command("upgrade", "Upgrades butler to the latest version").Alias("ugprade").Alias("update")
	ctx.Register(cmd, do)

	args.head = cmd.Flag("head", "Install bleeding-edge version").Bool()
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(ctx, *args.head))
}

func Do(ctx *mansion.Context, head bool) error {
	if head {
		if !comm.YesNo("Do you want to upgrade to the bleeding-edge version? Things may break!") {
			comm.Logf("Okay, not upgrading. Bye!")
			return nil
		}

		return applyUpgrade(ctx, "head", "head")
	}

	if ctx.Version == "head" {
		comm.Statf("Bleeding-edge, not upgrading unless told to.")
		comm.Logf("(Use `--head` if you want to upgrade to the latest bleeding-edge version)")
		return nil
	}

	comm.Opf("Looking for upgrades...")

	currentVer, latestVer, err := ctx.QueryLatestVersion()
	if err != nil {
		return fmt.Errorf("Version check failed: %s", err.Error())
	}

	if latestVer == nil || currentVer.GTE(*latestVer) {
		comm.Statf("Your butler is up-to-date. Have a nice day!")
		return nil
	}

	comm.Statf("Current version: %s", currentVer.String())
	comm.Statf("Latest version : %s", latestVer.String())

	if !comm.YesNo("Do you want to upgrade now?") {
		comm.Logf("Okay, not upgrading. Bye!")
		return nil
	}

	return applyUpgrade(ctx, currentVer.String(), latestVer.String())
}

func applyUpgrade(ctx *mansion.Context, before string, after string) error {
	execPath, err := osext.Executable()
	if err != nil {
		return err
	}

	oldPath := execPath + ".old"
	newPath := execPath + ".new"
	gzPath := newPath + ".gz"

	err = os.RemoveAll(newPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(gzPath)
	if err != nil {
		return err
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	fragment := fmt.Sprintf("v%s", after)
	if after == "head" {
		fragment = "head"
	}
	execURL := fmt.Sprintf("%s/%s/butler%s", ctx.UpdateBaseURL(), fragment, ext)

	gzURL := fmt.Sprintf("%s/%s/butler.gz", ctx.UpdateBaseURL(), fragment)
	comm.Opf("%s", gzURL)

	err = func() error {
		_, gErr := dl.Do(ctx, gzURL, gzPath)
		if gErr != nil {
			return gErr
		}

		fr, gErr := os.Open(gzPath)
		if gErr != nil {
			return gErr
		}
		defer fr.Close()

		gr, gErr := gzip.NewReader(fr)
		if gErr != nil {
			return gErr
		}

		fw, gErr := os.Create(newPath)
		if gErr != nil {
			return gErr
		}
		defer fw.Close()

		_, gErr = io.Copy(fw, gr)
		return gErr
	}()

	if err != nil {
		comm.Opf("Falling back to %s", execURL)
		_, err = dl.Do(ctx, execURL, newPath)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	err = os.Chmod(newPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	comm.Opf("Backing up current version to %s just in case...", oldPath)
	err = os.Rename(execPath, oldPath)
	if err != nil {
		return err
	}

	err = os.Rename(newPath, execPath)
	if err != nil {
		return err
	}

	err = os.Remove(oldPath)
	if err != nil {
		if os.IsPermission(err) && runtime.GOOS == "windows" {
			// poor windows doesn't like us removing executables from under it
			// I vote we move on and let butler.exe.old hang around.
		} else {
			return err
		}
	}

	comm.Statf("Upgraded butler from %s to %s. Have a nice day!", before, after)
	return nil
}
