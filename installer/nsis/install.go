package nsis

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/installer/bfs"

	"github.com/itchio/butler/cmd/elevate"
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
		cmd := []string{
			f.Name(),
			"/S",    // run the installer silently
			"/NCRC", // disable CRC-check, we do hash checking ourselves
		}

		pathArgs := getSeriouslyMisdesignedNsisPathArguments("/D=", params.InstallFolderPath)
		cmd = append(cmd, pathArgs...)

		consumer.Infof("launching nsis installer, command:")
		consumer.Infof("%s", strings.Join(cmd, " ::: "))

		elevateParams := &elevate.ElevateParams{
			Command: cmd,
			Stdout:  makeConsumerWriter(consumer, "out"),
			Stderr:  makeConsumerWriter(consumer, "err"),
		}

		ret, err := elevate.Elevate(elevateParams)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if ret != 0 {
			return fmt.Errorf("non-zero exit code %d (%x)", ret, ret)
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
