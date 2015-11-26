package counter

import "io"

type WriteCallback func(count int64)

type Counter struct {
	count  int64
	writer io.Writer

	onWrite WriteCallback
}

func New(writer io.Writer) *Counter {
	return &Counter{writer: writer}
}

func NewWithCallback(onWrite WriteCallback, writer io.Writer) *Counter {
	return &Counter{
		writer:  writer,
		onWrite: onWrite,
	}
}

func (w *Counter) Count() int64 {
	return w.count
}

func (w *Counter) Write(buffer []byte) (n int, err error) {
	if w.writer == nil {
		n = len(buffer)
	} else {
		n, err = w.writer.Write(buffer)
	}

	w.count += int64(n)
	if w.onWrite != nil {
		w.onWrite(w.count)
	}
	return
}

func (w *Counter) Close() error {
	return nil
}
