package inno

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

/*
 * InnoSetup docs: http://www.jrsoftware.org/ishelp/index.php?topic=setupcmdline
 */
func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	// we need the installer on disk to run it. this'll err if it's not,
	// and the caller is in charge of downloading it and calling us again.
	f, err := installer.AsLocalFile(params.File)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	angelParams := &bfs.SaveAngelsParams{
		Consumer: consumer,
		Folder:   params.InstallFolderPath,
		Receipt:  params.ReceiptIn,
	}

	cancel := make(chan struct{})
	defer close(cancel)
	bfs.StartAsymptoticProgress(consumer, cancel)

	angelResult, err := bfs.SaveAngels(angelParams, func() error {
		logPath := filepath.Join(params.StageFolderPath, "inno-install-log.txt")
		defer os.Remove(logPath)

		destPath := params.InstallFolderPath
		cmdTokens := []string{
			f.Name(),
			"/VERYSILENT",                    // run the installer silently
			"/SUPPRESSMSGBOXES",              // don't show any dialogs
			"/NOCANCEL",                      // no going back
			"/NORESTART",                     // prevent installer from restarting system
			fmt.Sprintf("/LOG=%s", logPath),  // store log on disk
			fmt.Sprintf("/DIR=%s", destPath), // specify install directory
		}

		consumer.Infof("→ Launching inno installer")

		// N.B: InnoSetup installers are smart enough to elevate themselves.
		exitCode, err := installer.RunCommand(consumer, cmdTokens)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = installer.CheckExitCode(exitCode, err)
		if err != nil {
			msg := messageForExitCode(exitCode)
			consumer.Warnf("InnoSetup installation failed: %s", msg)

			lf, openErr := os.Open(logPath)
			if openErr != nil {
				consumer.Warnf("...aditionally, we could not read the installation log: %s", openErr.Error())
			} else {
				defer lf.Close()
				var lines []string
				var maxLines = 20

				s := bufio.NewScanner(lf)
				for s.Scan() {
					lines = append(lines, s.Text())
					if len(lines) > maxLines {
						// this is extremely wasteful but hey, we don't care that much
						lines = lines[len(lines)-maxLines : len(lines)]
					}
				}
				consumer.Warnf("==== last %d lines of inno installation log ====", maxLines)
				for _, line := range lines {
					consumer.Warnf("%s", line)
				}
				consumer.Warnf("==== end of inno installation log ====")
			}

			if exitCodeIsAborted(exitCode) {
				return &buse.ErrAborted{}
			}

			return errors.Wrap(errors.New(msg), 0)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &installer.InstallResult{
		Files: angelResult.Files,
	}
	return res, nil
}

func exitCodeIsAborted(exitCode int) bool {
	// see http://www.jrsoftware.org/ishelp/index.php?topic=setupexitcodes
	switch exitCode {
	case 2:
		// the user clicked cancel or failed to allow elevation
		return true
	case 5:
		// the user clicked Cancel during the actual installation process, or chose Abort at an Abort-Retry-Ignore box.
		return true
	default:
		return false
	}
}

func messageForExitCode(exitCode int) string {
	// see http://www.jrsoftware.org/ishelp/index.php?topic=setupexitcodes
	switch exitCode {
	case 1:
		return `Setup failed to initialize`
	case 2:
		return `The user clicked Cancel in the wizard before the actual installation started, or chose "No" on the opening "This will install..." message box.`
	case 3:
		return `A fatal error occurred while preparing to move to the next installation phase (for example, from displaying the pre-installation wizard pages to the actual installation process). This should never happen except under the most unusual of circumstances, such as running out of memory or Windows resources.`
	case 4:
		return `A fatal error occurred during the actual installation process.`
	case 5:
		return `The user clicked Cancel during the actual installation process, or chose Abort at an Abort-Retry-Ignore box.`
	case 6:
		return `The Setup process was forcefully terminated by the debugger (Run | Terminate was used in the IDE).`
	case 7:
		return `The Preparing to Install stage determined that Setup cannot proceed with installation.`
	case 8:
		return `The Preparing to Install stage determined that Setup cannot proceed with installation, and that the system needs to be restarted in order to correct the problem.`
	default:
		return fmt.Sprintf(`Unknown error (exit code %d)`, exitCode)
	}
}
