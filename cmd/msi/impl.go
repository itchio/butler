package msi

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var showFullMsiLog = os.Getenv("BUTLER_FULL_MSI_LOG") == "1"

func Install(consumer *state.Consumer, msiPath string, logPathIn string, onError MSIErrorCallback) error {
	return withMsiLogging(consumer, logPathIn, func(logPath string) error {
		cmd := exec.Command("msiexec", "/i", msiPath, "/qn", "/l*v", logPath)
		err := cmd.Run()
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}, onError)
}

func Uninstall(consumer *state.Consumer, msiPath string, logPathIn string, onError MSIErrorCallback) error {
	return withMsiLogging(consumer, logPathIn, func(logPath string) error {
		cmd := exec.Command("msiexec", "/x", msiPath, "/qn", "/l*v", logPath)
		err := cmd.Run()
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}, onError)
}

type MSITaskFunc func(logPath string) error

func withMsiLogging(consumer *state.Consumer, logPath string, task MSITaskFunc, onError MSIErrorCallback) error {
	if logPath == "" {
		tempDir, err := ioutil.TempDir("", "butler-msi-logs")
		if err != nil {
			return errors.WithStack(err)
		}

		defer func() {
			os.RemoveAll(tempDir)
		}()

		logPath = filepath.Join(tempDir, "msi-install-log.txt")
	}

	taskErr := task(logPath)

	if taskErr != nil {
		consumer.Infof("")

		lf, openErr := os.Open(logPath)
		if openErr != nil {
			consumer.Warnf("Can't open MSI log: %s", openErr.Error())
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
