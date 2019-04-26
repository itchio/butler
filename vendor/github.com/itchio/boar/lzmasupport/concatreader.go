package lzmasupport

import "io"

type concatReader struct {
	readers []io.Reader
}

func (cr *concatReader) popReader() {
	cr.readers = cr.readers[1:]
}

func (cr *concatReader) Read(p []byte) (int, error) {
	if len(cr.readers) == 0 {
		return 0, io.EOF
	}

	reader := cr.readers[0]
	n, err := reader.Read(p)
	if err == io.EOF {
		cr.popReader()

		// some readers return the last bytes *and* io.EOF
		if n > 0 {
			// if this was the last reader, we'll return io.EOF on next call.
			// if not, this is a short read and we'll read from
			// the next reader next call.
			return n, nil
		} else {
			// if this was the last reader, this returns EOF immediately
			// if not, it reads from the next reader
			return cr.Read(p)
		}
	}
	// any lack of error (even when doing a short read), or
	// any error besides io.EOF, must be forwarded
	return n, err
}
