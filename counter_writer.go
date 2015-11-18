package main

import "io"

type counterWriter struct {
	count  int64
	writer io.Writer
}

func (w *counterWriter) Write(buffer []byte) (int, error) {
	if w.writer != nil {
		written, err := w.writer.Write(buffer)
		w.count += int64(written)
		return written, err
	}

	w.count += int64(len(buffer))
	return len(buffer), nil
}

func (w *counterWriter) Close() error {
	if w.writer != nil {
		if v, ok := w.writer.(io.Closer); ok {
			return v.Close()
		}
	}

	return nil
}
