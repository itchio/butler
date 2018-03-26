package nsis

import (
	"github.com/itchio/butler/installer/bfs"
	"github.com/pkg/errors"

	"github.com/itchio/butler/installer"
)

/*
 * Install performs installation for an NSIS package.
 *
 * NSIS docs: http://nsis.sourceforge.net/Docs/Chapter3.html
 * When ran without elevate, some NSIS installers will silently fail.
 * So, we run them with elevate all the time.
 */
func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	// we need the installer on disk to run it. this'll err if it's not,
	// and the caller is in charge of downloading it and calling us again.
	f, err := installer.AsLocalFile(params.File)
	if err != nil {
		return nil, errors.WithStack(err)
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
		cmdTokens := []string{
			f.Name(),
			"/S",    // run the installer silently
			"/NCRC", // disable CRC-check, we do hash checking ourselves
		}

		pathArgs := getSeriouslyMisdesignedNsisPathArguments("/D=", params.InstallFolderPath)
		cmdTokens = append(cmdTokens, pathArgs...)

		consumer.Infof("→ Launching nsis installer")

		exitCode, err := installer.RunElevatedCommand(consumer, cmdTokens)
		err = installer.CheckExitCode(exitCode, err)
		if err != nil {
			return errors.Wrap(err, "making sure nsis installer ran correctly")
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "running nsis installer")
	}

	res := &installer.InstallResult{
		Files: angelResult.Files,
	}
	return res, nil
}
