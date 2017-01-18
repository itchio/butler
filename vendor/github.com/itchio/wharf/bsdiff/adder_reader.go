package bsdiff

import "io"

type AdderReader struct {
	Buffer []byte
	Reader io.Reader

	offset int
}

var _ io.Reader = (*AdderReader)(nil)

func (ar *AdderReader) Read(p []byte) (int, error) {
	n, err := ar.Reader.Read(p)
	if err != nil {
		return n, err
	}

	for i := range p {
		p[i] += ar.Buffer[ar.offset]
		ar.offset++
	}

	return n, nil
}
