package dash

import (
	"bufio"
	"io"
	"regexp"
)

func sniffLove(r io.ReadSeeker, size int64, path string) (*Candidate, error) {
	res := &Candidate{
		Flavor:   FlavorLove,
		Path:     path,
		LoveInfo: &LoveInfo{},
	}

	s := bufio.NewScanner(r)

	re := regexp.MustCompile(`t\.version\s*=\s*"([^"]+)"`)

	for s.Scan() {
		line := s.Bytes()
		matches := re.FindSubmatch(line)
		if len(matches) == 2 {
			res.LoveInfo.Version = string(matches[1])
			break
		}
	}

	return res, nil
}
