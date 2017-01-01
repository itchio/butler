package onclose

import "io"

// An CloseCallback is a callback type called when a Writer is closed
type CloseCallback func()

// Writer implements io.WriteCloser and has callbacks for before and after the
// underlying writer is closed
type Writer struct {
	Writer      io.Writer
	BeforeClose CloseCallback
	AfterClose  CloseCallback
}

var _ io.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)

// Write relays the write directly to the underlying writer
func (w *Writer) Write(buf []byte) (int, error) {
	return w.Writer.Write(buf)
}

// Close first calls OnClose, then closes the underlying writer
// (if the callback didn't return an error)
func (w *Writer) Close() error {
	if w.BeforeClose != nil {
		w.BeforeClose()
	}

	if closer, ok := w.Writer.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return err
		}
	}

	if w.AfterClose != nil {
		w.AfterClose()
	}

	return nil
}
