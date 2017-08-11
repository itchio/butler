package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

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

type MSIWindowsInstallerError struct {
	Code int64  `json:"code"`
	Text string `json:"text"`
}

type MSIWindowsInstallerErrorResult struct {
	Type  string                   `json:"type"`
	Value MSIWindowsInstallerError `json:"value"`
}

func msiInstall(msiPath string, logPath string, target string) {
	must(doMsiInstall(msiPath, logPath, target))
}

func doMsiInstall(msiPath string, logPathIn string, target string) error {
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

	return withMsiLogging(logPathIn, func() error {
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
	})

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

	err := withMsiLogging("", func() error {
		return gowin32.UninstallProduct(productCode)
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Statf("Uninstalled in %s", time.Since(startTime))

	return nil
}

type MsiLogCallback func() error

func withMsiLogging(logPath string, f MsiLogCallback) error {
	if logPath == "" {
		tempDir, err := ioutil.TempDir("", "butler-msi-logs")
		if err != nil {
			return errors.Wrap(err, 0)
		}

		defer func() {
			os.RemoveAll(tempDir)
		}()

		logPath = filepath.Join(tempDir, "msi-install-log.txt")
	}

	{
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

	taskErr := f()

	if taskErr != nil {
		comm.Logf("")

		lf, openErr := os.Open(logPath)
		if openErr != nil {
			comm.Warnf("And what's more, we can't open the log: %s", openErr.Error())
		} else {
			// grok UTF-16
			win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
			// ...but abide by the BOM if there's one
			utf16bom := unicode.BOMOverride(win16be.NewDecoder())

			unicodeReader := transform.NewReader(lf, utf16bom)

			defer lf.Close()
			s := bufio.NewScanner(unicodeReader)

			var lines []string
			for s.Scan() {
				lines = append(lines, s.Text())
			}

			if *appArgs.verbose {
				comm.Logf("Full log (run without verbose mode to get only errors): ")
				for _, line := range lines {
					comm.Logf("%s", line)
				}
			} else {
				// leading (?i) = case-insensitive in golang
				// We're looking for lines like:
				// [blah] [time] Error 1925. You don't have enough permissions for nice things. This is why you can't have nice things.
				re := regexp.MustCompile(`Error ([0-9]+)\..*`)

				var errors []string

				for _, line := range lines {
					submatch := re.FindStringSubmatch(line)
					if len(submatch) > 0 {
						duplicate := false

						for _, e := range errors {
							if e == submatch[0] {
								// already got it, abort!
								duplicate = true
								break
							}
						}

						if !duplicate {
							text := submatch[0]
							codeString := submatch[1]

							errors = append(errors, text)

							code, err := strconv.ParseInt(codeString, 10, 64)
							if err != nil {
								comm.Debugf("Couldn't parse error code '%s'", codeString)
							} else {
								comm.Result(&MSIWindowsInstallerErrorResult{
									Type: "windowsInstallerError",
									Value: MSIWindowsInstallerError{
										Code: code,
										Text: text,
									},
								})
							}
						}
					}
				}

				if len(errors) > 0 {
					for _, e := range errors {
						comm.Logf("  %s", e)
					}
				} else {
					comm.Logf("Full MSI log: ")
					for _, line := range lines {
						comm.Logf("%s", line)
					}
				}
			}
			if scanErr := s.Err(); scanErr != nil {
				comm.Warnf("While reading msi log: %s", scanErr.Error())
			}
		}

		comm.Logf("")
		return fmt.Errorf("%s", taskErr.Error())
	}

	return nil
}
