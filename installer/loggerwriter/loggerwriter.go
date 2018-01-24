package loggerwriter

import (
	"bufio"
	"io"

	"github.com/itchio/wharf/state"
)

// New returns an io.Writer that, when a line is
// written to, writes it as a log message to the consumer with the
// given prefix
func New(consumer *state.Consumer, prefix string) io.Writer {
	pr, pw := io.Pipe()

	go func() {
		// note: we don't care terribly about bufio.Scanner error
		// conditions for this.
		s := bufio.NewScanner(pr)

		for s.Scan() {
			if prefix == "err" {
				consumer.Warnf("[%s] %s", prefix, s.Text())
			} else {
				consumer.Infof("[%s] %s", prefix, s.Text())
			}
		}
	}()

	return pw
}
