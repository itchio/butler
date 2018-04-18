package dash

import (
	"bufio"
	"io"
	"strings"
)

func sniffScript(r io.ReadSeeker, size int64) (*Candidate, error) {
	res := &Candidate{
		Flavor:     FlavorScript,
		ScriptInfo: &ScriptInfo{},
	}

	_, err := r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(r)

	if s.Scan() {
		line := s.Text()
		if len(line) > 2 {
			// skip over the shebang
			res.ScriptInfo.Interpreter = strings.TrimSpace(line[2:])
		}
	}

	return res, nil
}
