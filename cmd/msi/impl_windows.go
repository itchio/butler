// +build windows

package msi

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

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"github.com/winlabs/gowin32"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var showFullMsiLog = os.Getenv("BUTLER_FULL_MSI_LOG") == "1"

func Info(consumer *state.Consumer, msiPath string) (*MSIInfoResult, error) {
	initMsi()

	msiPath, err := filepath.Abs(msiPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	pkg, err := gowin32.OpenInstallerPackage(msiPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	defer pkg.Close()

	productCode, err := pkg.GetProductProperty("ProductCode")
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	consumer.Debugf("Product code for %s: %s", msiPath, productCode)

	state := gowin32.GetInstalledProductState(productCode)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer.Debugf("Installed product state: %s", installStateToString(state))

	res := &MSIInfoResult{
		ProductCode:  productCode,
		InstallState: installStateToString(state),
	}

	if state == gowin32.InstallStateDefault {
		installLocation, err := gowin32.GetInstalledProductProperty(productCode, gowin32.InstallPropertyInstallLocation)
		if err != nil {
			consumer.Debugf("Could not get install location: %s", err.Error())
		} else {
			res.InstallLocation = installLocation
		}
	}

	return res, nil
}

func ProductInfo(consumer *state.Consumer, productCode string) (*MSIInfoResult, error) {
	initMsi()

	state := gowin32.GetInstalledProductState(productCode)

	res := &MSIInfoResult{
		ProductCode:  productCode,
		InstallState: installStateToString(state),
	}
	return res, nil
}

func Install(consumer *state.Consumer, msiPath string, logPathIn string, target string, onError MSIErrorCallback) error {
	initMsi()

	startTime := time.Now()

	msiPath, err := filepath.Abs(msiPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer.Debugf("Assessing state of %s", msiPath)

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
		consumer.Debugf("Product code for %s: %s", msiPath, productCode)
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
		consumer.Opf("Already installed, repairing from %s", msiPath)
		repair = true
	} else {
		consumer.Opf("Installing %s", msiPath)
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
		// transform inputs like
		//   - `./dir`
		//   - `C:/msys64/home/john/dir`
		// into:
		//   - `C:\msys64\home\john\dir`
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		// throw everything we got to try and get a local install
		commandLine += " ALLUSERS=2 MSIINSTALLPERUSER=1"
		commandLine += fmt.Sprintf(" TARGETDIR=\"%s\" INSTALLDIR=\"%s\" APPDIR=\"%s\"", absTarget, absTarget, absTarget)
		consumer.Infof("...will install in folder %s", absTarget)
	}

	consumer.Debugf("Final command line: %s", commandLine)

	return withMsiLogging(consumer, logPathIn, func() error {
		if repair {
			ilvl := gowin32.InstallLevelDefault
			istate := gowin32.InstallStateDefault
			err := gowin32.ConfigureInstalledProduct(productCode, ilvl, istate, commandLine)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			consumer.Statf("Repaired in %s", time.Since(startTime))
		} else {
			err := gowin32.InstallProduct(msiPath, commandLine)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			consumer.Statf("Installed in %s", time.Since(startTime))
		}
		return nil
	}, onError)
}

func Uninstall(consumer *state.Consumer, productCode string, onError MSIErrorCallback) error {
	initMsi()

	if !strings.HasPrefix(productCode, "{") {
		consumer.Logf("Argument doesn't look like a product ID, treating it like an MSI file")

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

	consumer.Opf("Uninstalling product %s", productCode)

	startTime := time.Now()

	return withMsiLogging(consumer, "", func() error {
		err := gowin32.UninstallProduct(productCode)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		consumer.Statf("Uninstalled in %s", time.Since(startTime))
		return nil
	}, onError)
}

type MSITaskFunc func() error

func withMsiLogging(consumer *state.Consumer, logPath string, task MSITaskFunc, onError MSIErrorCallback) error {
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

		consumer.Debugf("...will write log to %s", logPath)
	}

	taskErr := task()

	if taskErr != nil {
		consumer.Infof("")

		lf, openErr := os.Open(logPath)
		if openErr != nil {
			consumer.Warnf("And what's more, we can't open the log: %s", openErr.Error())
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

			if showFullMsiLog {
				consumer.Infof("Full log (run without verbose mode to get only errors): ")
				for _, line := range lines {
					consumer.Infof("%s", line)
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
								consumer.Debugf("Couldn't parse error code '%s'", codeString)
							} else {
								if onError != nil {
									onError(MSIWindowsInstallerError{
										Code: code,
										Text: text,
									})
								}
							}
						}
					}
				}

				if len(errors) > 0 {
					for _, e := range errors {
						consumer.Infof("  %s", e)
					}
				} else {
					consumer.Infof("Full MSI log: ")
					for _, line := range lines {
						consumer.Infof("%s", line)
					}
				}
			}
			if scanErr := s.Err(); scanErr != nil {
				consumer.Warnf("While reading msi log: %s", scanErr.Error())
			}
		}

		consumer.Logf("")
		return fmt.Errorf("%s", taskErr.Error())
	}

	return nil
}
