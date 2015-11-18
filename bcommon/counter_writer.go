package bcommon

import "io"

type CounterWriter struct {
	count  int64
	writer io.Writer
}

func NewCounterWriter(writer io.Writer) *CounterWriter {
	return &CounterWriter{writer: writer}
}

func (w *CounterWriter) Count() int64 {
	return w.count
}

func (w *CounterWriter) Write(buffer []byte) (int, error) {
	if w.writer != nil {
		written, err := w.writer.Write(buffer)
		w.count += int64(written)
		return written, err
	}

	w.count += int64(len(buffer))
	return len(buffer), nil
}

func (w *CounterWriter) Close() error {
	if w.writer != nil {
		if v, ok := w.writer.(io.Closer); ok {
			return v.Close()
		}
	}

	return nil
}
