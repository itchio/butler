package lzmasupport

import "io"

type concatReader struct {
	readers []io.Reader
	index   int
}

func (cr *concatReader) Read(buf []byte) (int, error) {
	n, err := cr.readers[cr.index].Read(buf)
	for err != nil || n < len(buf) {
		if err == io.EOF {
			if cr.index == len(cr.readers)-1 {
				return n, err
			}
			cr.index++
			err = nil
		}

		var m int
		m, err = cr.readers[cr.index].Read(buf[n:])
		n += m
	}
	return n, err
}
