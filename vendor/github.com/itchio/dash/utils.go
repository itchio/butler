package dash

import (
	"io"
	"strings"
)

func spellHas(spell []string, token string) bool {
	for _, tok := range spell {
		if tok == token {
			return true
		}
	}
	return false
}

func pathDepth(path string) int {
	return len(strings.Split(path, "/"))
}

func hasExt(path string, ext string) bool {
	return strings.HasSuffix(strings.ToLower(path), ext)
}

// Adapt an io.ReadSeeker into an io.ReaderAt in the dumbest possible fashion

type readerAtFromSeeker struct {
	rs io.ReadSeeker
}

var _ io.ReaderAt = (*readerAtFromSeeker)(nil)

func (r *readerAtFromSeeker) ReadAt(b []byte, off int64) (int, error) {
	_, err := r.rs.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return r.rs.Read(b)
}

func selectByFlavor(candidates []*Candidate, f Flavor) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if c.Flavor == f {
			res = append(res, c)
		}
	}
	return res
}

func selectByArch(candidates []*Candidate, a Arch) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if c.Arch == a {
			res = append(res, c)
		}
	}
	return res
}

type candidateFilter func(candidate *Candidate) bool

func selectByFunc(candidates []*Candidate, f candidateFilter) []*Candidate {
	res := make([]*Candidate, 0)
	for _, c := range candidates {
		if f(c) {
			res = append(res, c)
		}
	}
	return res
}
