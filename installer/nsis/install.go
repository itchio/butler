package nsis

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/installer/bfs"
	"github.com/itchio/wharf/state"

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
	// we need the installer on disk to run it. this'll err if it's not,
	// and the caller is in charge of downloading it and calling us again.
	f, err := installer.AsLocalFile(params.File)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	angelParams := &bfs.SaveAngelsParams{
		Consumer: params.Consumer,
		Folder:   params.InstallFolderPath,
	}

	angelResult, err := bfs.SaveAngels(angelParams, func() error {
		stats, err := f.Stat()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		cmd := []string{
			stats.Name(),
			"/S",    // run the installer silently
			"/NCRC", // disable CRC-check, we do hash checking ourselves
		}

		pathArgs := getSeriouslyMisdesignedNsisPathArguments("/D=", params.InstallFolderPath)
		cmd = append(cmd, pathArgs...)

		elevateParams := &elevate.ElevateParams{
			Command: cmd,
			Stdout:  makeConsumerWriter(params.Consumer, "out"),
			Stderr:  makeConsumerWriter(params.Consumer, "err"),
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

/*
 * Returns an array of arguments that will make an NSIS installer or uninstaller happy
 *
 * The docs say to "not wrap the argument in double quotes" but what they really mean is
 * just pass it as separate arguments (due to how f*cked argument parsing is)
 *
 * So this takes `/D=`, `C:\Itch Games\something` and returns
 * [`/D=C:\Itch`, `Games\something`]
 *
 * @param prefix something like `/D=` or `_?=` probably
 * @param path a path, may contain spaces, may not
 */
func getSeriouslyMisdesignedNsisPathArguments(prefix string, name string) []string {
	tokens := strings.Split(name, " ")
	tokens[0] = fmt.Sprintf("%s%s", prefix, tokens[0])
	return tokens
}

// makeConsumerWriter returns an io.Writer that, when a line is
// written to, writes it as a log message to the consumer with the
// given prefix
func makeConsumerWriter(consumer *state.Consumer, prefix string) io.Writer {
	pr, pw := io.Pipe()

	go func() {
		// note: we don't care terribly about bufio.Scanner error
		// conditions for this.
		s := bufio.NewScanner(pr)

		for s.Scan() {
			if prefix == "err" {
				consumer.Warnf("[%s] %s", s.Text())
			} else {
				consumer.Infof("[%s] %s", s.Text())
			}
		}
	}()

	return pw
}
