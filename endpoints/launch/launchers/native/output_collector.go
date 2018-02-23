package native

import (
	"bufio"
	"io"
)

type outputCollector struct {
	lines  []string
	writer io.Writer
}

var _ io.Writer = (*outputCollector)(nil)

func newOutputCollector(maxLines int) *outputCollector {
	pipeR, pipeW := io.Pipe()

	oc := &outputCollector{
		writer: pipeW,
	}

	go func() {
		s := bufio.NewScanner(pipeR)
		for s.Scan() {
			line := s.Text()
			oc.lines = append(oc.lines, line)

			if len(oc.lines) > maxLines {
				oc.lines = oc.lines[1:]
			}
		}
	}()

	return oc
}

func (oc *outputCollector) Lines() []string {
	return oc.lines
}

func (oc *outputCollector) Write(p []byte) (int, error) {
	return oc.writer.Write(p)
}
