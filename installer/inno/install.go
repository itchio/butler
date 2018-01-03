package inno

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
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

		consumer.Infof("launching inno installer")

		// N.B: InnoSetup installers are smart enough to elevate themselves.
		exitCode, err := installer.RunCommand(consumer, cmdTokens)
		err = installer.CheckExitCode(exitCode, err)
		if err != nil {
			consumer.Warnf("installation failed: %s", err.Error())

			lf, err := os.Open(logPath)
			if err != nil {
				consumer.Warnf("...aditionally, we could not read the installation log: %s", err.Error())
			} else {
				defer lf.Close()
				consumer.Warnf("==== inno installation log start ====")
				s := bufio.NewScanner(lf)
				for s.Scan() {
					consumer.Warnf(s.Text())
				}
				consumer.Warnf("==== inno installation log end ====")
			}

			return errors.Wrap(err, 0)
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
