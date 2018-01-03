package nsis

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/itchio/wharf/state"
)

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
