package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/winlabs/gowin32"
)

func msiInfo(msiPath string) {
	must(doMsiInfo(msiPath))
}

/**
 * MSIInfoResult describes an MSI package's properties
 */
type MSIInfoResult struct {
	ProductCode  string `json:"productCode"`
	InstallState string `json:"installState"`
}

func doMsiInfo(msiPath string) error {
	initMsi()

	msiPath, err := filepath.Abs(msiPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	pkg, err := gowin32.OpenInstallerPackage(msiPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	defer pkg.Close()

	productCode, err := pkg.GetProductProperty("ProductCode")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	comm.Debugf("Product code for %s: %s", msiPath, productCode)

	state := gowin32.GetInstalledProductState(productCode)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Debugf("Installed product state: %s", installStateToString(state))

	if *appArgs.json {
		comm.Result(&MSIInfoResult{
			ProductCode:  productCode,
			InstallState: installStateToString(state),
		})
	} else {
		comm.Statf("MSI product code: %s", productCode)
		comm.Statf("Install state: %s", installStateToString(state))
	}

	return nil
}

func msiProductInfo(productCode string) {
	must(doMsiProductInfo(productCode))
}

func doMsiProductInfo(productCode string) error {
	initMsi()

	state := gowin32.GetInstalledProductState(productCode)

	comm.Logf("Installed product state: %s", installStateToString(state))

	if *appArgs.json {
		comm.Result(&MSIInfoResult{
			ProductCode:  productCode,
			InstallState: installStateToString(state),
		})
	}

	return nil
}

func msiInstall(msiPath string, logPath string, target string) {
	must(doMsiInstall(msiPath, logPath, target))
}

func doMsiInstall(msiPath string, logPath string, target string) error {
	initMsi()

	startTime := time.Now()

	msiPath, err := filepath.Abs(msiPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Debugf("Assessing state of %s", msiPath)

	var productCode string

	err = func() error {
		pkg, err := gowin32.OpenInstallerPackage(msiPath)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		defer pkg.Close()

		productCode, err = pkg.GetProductProperty("ProductCode")
		if err != nil {
			return errors.Wrap(err, 0)
		}
		comm.Debugf("Product code for %s: %s", msiPath, productCode)
		return nil
	}()
	if err != nil {
		return err
	}

	state := gowin32.GetInstalledProductState(productCode)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	repair := false
	if state == gowin32.InstallStateDefault {
		comm.Opf("Already installed, repairing from %s", msiPath)
		repair = true
	} else {
		comm.Opf("Installing %s", msiPath)
	}

	if logPath != "" {
		// equivalent to "/lv"
		logMode := gowin32.InstallLogModeVerbose
		logAttr := gowin32.InstallLogAttributesFlushEachLine
		err := gowin32.EnableInstallerLog(logMode, logPath, logAttr)
		if err != nil {
			return errors.Wrap(err, 0)
		}
		defer func() {
			gowin32.DisableInstallerLog()
		}()

		comm.Debugf("...will write log to %s", logPath)
	}

	// s = Recreate all shortcuts
	// m = Rewrite all HKLM and HKCR registry entries
	// u = Rewrite all HKCU and HKU registry entries
	// p = Reinstall only if the file is missing
	// We can't use "o", "e", "d", "c" (reinstall older, equal-or-older, different, checksum)
	// because they require the source package to be available
	// (which isn't always true, see https://github.com/itchio/itch/issues/1304)
	// We won't use "a" (reinstall all) because it's overkill.
	commandLine := "REINSTALLMODE=smup REBOOT=reallysuppress"

	if target != "" {
		// throw everything we got to try and get a local install
		commandLine += " ALLUSERS=2 MSIINSTALLPERUSER=1"
		commandLine += fmt.Sprintf(" TARGETDIR=\"%s\" INSTALLDIR=\"%s\" APPDIR=\"%s\"", target, target, target)
		comm.Debugf("...will install in folder %s", target)
	}

	comm.Debugf("Final command line: %s", commandLine)

	if repair {
		ilvl := gowin32.InstallLevelDefault
		istate := gowin32.InstallStateDefault
		err = gowin32.ConfigureInstalledProduct(productCode, ilvl, istate, commandLine)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.Opf("Repaired in %s", time.Since(startTime))
	} else {
		err = gowin32.InstallProduct(msiPath, commandLine)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		comm.Opf("Installed in %s", time.Since(startTime))
	}

	return nil
}

func msiUninstall(productCode string) {
	must(doMsiUninstall(productCode))
}

func doMsiUninstall(productCode string) error {
	initMsi()

	if !strings.HasPrefix(productCode, "{") {
		comm.Logf("Argument doesn't look like a product ID, treating it like an MSI file")

		err := func() error {
			msiPath, err := filepath.Abs(productCode)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			pkg, err := gowin32.OpenInstallerPackage(msiPath)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			defer pkg.Close()

			fileProductCode, err := pkg.GetProductProperty("ProductCode")
			if err != nil {
				return errors.Wrap(err, 0)
			}
			productCode = fileProductCode
			return nil
		}()
		if err != nil {
			return err
		}
	}

	comm.Opf("Uninstalling product %s", productCode)

	startTime := time.Now()

	err := gowin32.UninstallProduct(productCode)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("Uninstalled in %s", time.Since(startTime))

	return nil
}
