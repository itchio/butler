// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package brotli

import "io"

type writer struct {
	InputOffset  int64 // Total number of bytes issued to Write
	OutputOffset int64 // Total number of bytes written to underlying io.Writer

	wr  bitWriter // Output destination
	err error     // Persistent error
}

func newWriter(w io.Writer) *writer {
	return nil
}

func (bw *writer) Write(buf []byte) (int, error) {
	return 0, nil
}

func (bw *writer) Close() error {
	return nil
}

func (bw *writer) Reset(w io.Writer) {
	return
}
