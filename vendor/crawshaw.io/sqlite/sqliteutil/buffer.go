// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package sqliteutil

import (
	"errors"
	"io"

	"crawshaw.io/sqlite"
)

// A Buffer is a variable-sized bytes buffer backed by SQLite blobs.
//
// The bytes are broken into pages, with the first and last pages
// stored in memory, and intermediate pages loaded into blobs.
// Unlike a single SQLite blob, a Buffer can grow beyond its initial size.
// The blobs are allocated in a temporary table.
//
// A Buffer is very similar to a bytes.Buffer.
type Buffer struct {
	io.Reader
	io.Writer
	io.ByteScanner

	err  error
	conn *sqlite.Conn

	// cap(rbuf) == cap(wbuf) == blobs[N].Size()

	rbuf  []byte  // read buffer
	roff  int     // read head position in roff
	blobs []tblob // blobs storing data between rbuf and wbuf
	wbuf  []byte  // write buffer

	freelist []tblob
}

type tblob struct {
	blob  *sqlite.Blob
	rowid int64
}

// NewBuffer creates a Buffer with 16KB pages.
func NewBuffer(conn *sqlite.Conn) (*Buffer, error) {
	return NewBufferSize(conn, 16*1024)
}

// NewBufferSize creates a Buffer with a specified page size.
func NewBufferSize(conn *sqlite.Conn, pageSize int) (*Buffer, error) {
	bb := &Buffer{
		conn: conn,
		rbuf: make([]byte, 0, pageSize),
		wbuf: make([]byte, 0, pageSize),
	}
	stmt := conn.Prep("CREATE TEMP TABLE IF NOT EXISTS BlobBuffer (blob BLOB);")
	if _, err := stmt.Step(); err != nil {
		return nil, err
	}
	return bb, nil
}

func (bb *Buffer) alloc() (tblob, error) {
	if len(bb.freelist) > 0 {
		b := bb.freelist[len(bb.freelist)-1]
		bb.freelist = bb.freelist[:len(bb.freelist)-1]
		return b, nil
	}

	stmt := bb.conn.Prep("INSERT INTO BlobBuffer (blob) VALUES ($blob);")
	stmt.SetZeroBlob("$blob", int64(len(bb.rbuf)))
	if _, err := stmt.Step(); err != nil {
		return tblob{}, err
	}
	rowid := bb.conn.LastInsertRowID()
	blob, err := bb.conn.OpenBlob("temp", "BlobBuffer", "blob", rowid, true)
	if err != nil {
		return tblob{}, err
	}
	return tblob{
		blob:  blob,
		rowid: rowid,
	}, nil
}

func (bb *Buffer) free(b tblob) {
	bb.freelist = append(bb.freelist, b)
}

func (bb *Buffer) wbufEnsureSpace() error {
	if len(bb.wbuf) < cap(bb.wbuf) {
		return nil
	}

	// Flush the write buffer.
	if len(bb.blobs) == 0 && bb.roff == len(bb.rbuf) {
		// Short cut. The write buffer is full, but
		// there are no on-disk blobs and the read
		// buffer is empty. So push these bytes
		// directly to the front of the Buffer.
		bb.rbuf, bb.wbuf = bb.wbuf, bb.rbuf[:0]
		bb.roff = 0
	} else {
		tblob, err := bb.alloc()
		if err != nil {
			bb.err = err
			return err
		}
		if _, err := tblob.blob.WriteAt(bb.wbuf, 0); err != nil {
			bb.err = err
			return err
		}
		bb.blobs = append(bb.blobs, tblob)
		bb.wbuf = bb.wbuf[:0]
	}

	return nil
}

// WriteByte appends a byte to the buffer, growing it as needed.
func (bb *Buffer) WriteByte(c byte) error {
	if bb.err != nil {
		return bb.err
	}
	if err := bb.wbufEnsureSpace(); err != nil {
		return err
	}
	bb.wbuf = append(bb.wbuf, c)
	return nil
}

func (bb *Buffer) UnreadByte() error {
	if bb.err != nil {
		return bb.err
	}
	if bb.roff == 0 {
		return errors.New("sqliteutil.Buffer: UnreadByte: no byte to unread")
	}
	bb.roff--
	return nil
}

func (bb *Buffer) Write(p []byte) (n int, err error) {
	if bb.err != nil {
		return 0, bb.err
	}

	for len(p) > 0 {
		if err := bb.wbufEnsureSpace(); err != nil {
			return n, err
		}

		// TODO: shortcut for writing large p directly into a new blob

		nn := len(p)
		if rem := cap(bb.wbuf) - len(bb.wbuf); nn > rem {
			nn = rem
		}
		bb.wbuf = append(bb.wbuf, p[:nn]...) // never grows wbuf
		n += nn
		p = p[nn:]
	}

	return n, nil
}

func (bb *Buffer) WriteString(p string) (n int, err error) {
	if bb.err != nil {
		return 0, bb.err
	}

	for len(p) > 0 {
		if err := bb.wbufEnsureSpace(); err != nil {
			return n, err
		}

		// TODO: shortcut for writing large p directly into a new blob

		nn := len(p)
		if rem := cap(bb.wbuf) - len(bb.wbuf); nn > rem {
			nn = rem
		}
		bb.wbuf = append(bb.wbuf, p[:nn]...) // never grows wbuf
		n += nn
		p = p[nn:]
	}

	return n, nil
}

func (bb *Buffer) rbufFill() error {
	if bb.roff < len(bb.rbuf) {
		return nil
	}

	// Read buffer is empty. Fill it.
	if len(bb.blobs) > 0 {
		// Read the first blob entirely into the read buffer.
		// TODO: shortcut for if len(p) >= blob.Size()
		bb.roff = 0
		bb.rbuf = bb.rbuf[:cap(bb.rbuf)]

		tblob := bb.blobs[0]
		bb.blobs = bb.blobs[1:]
		if nn, err := tblob.blob.ReadAt(bb.rbuf, 0); err != nil {
			bb.err = err
			return err
		} else if nn != len(bb.rbuf) {
			panic("sqliteutil.Buffer: short read from blob")
		}
		bb.free(tblob)
		return nil
	}
	if len(bb.wbuf) > 0 {
		// No blobs. Swap the write buffer bytes here directly.
		bb.rbuf, bb.wbuf = bb.wbuf, bb.rbuf[:0]
		bb.roff = 0
	}

	if bb.roff == len(bb.rbuf) {
		return io.EOF
	}
	return nil
}

func (bb *Buffer) ReadByte() (byte, error) {
	if bb.err != nil {
		return 0, bb.err
	}
	if err := bb.rbufFill(); err != nil {
		return 0, err
	}
	c := bb.rbuf[bb.roff]
	bb.roff++
	return c, nil
}

func (bb *Buffer) Read(p []byte) (n int, err error) {
	if bb.err != nil {
		return 0, bb.err
	}
	if err := bb.rbufFill(); err != nil {
		return 0, err
	}
	if bb.roff == len(bb.rbuf) {
		return 0, io.EOF
	}

	n = copy(p, bb.rbuf[bb.roff:])
	bb.roff += n
	return n, nil
}

func (bb *Buffer) Len() (n int64) {
	n = int64(len(bb.rbuf) - bb.roff)
	n += int64(cap(bb.rbuf) * len(bb.blobs))
	n += int64(len(bb.wbuf))
	return n
}

func (bb *Buffer) Cap() (n int64) {
	pageSize := int64(cap(bb.rbuf))
	return (2 + int64(len(bb.blobs)+len(bb.freelist))) * pageSize
}

func (bb *Buffer) Reset() {
	bb.rbuf = bb.rbuf[:0]
	bb.wbuf = bb.wbuf[:0]
	bb.roff = 0
	bb.freelist = append(bb.freelist, bb.blobs...)
	bb.blobs = nil
}

func (bb *Buffer) Close() error {
	close := func(tblob tblob) {
		err := tblob.blob.Close()
		if bb.err == nil {
			bb.err = err
		}
	}
	for _, tblob := range bb.blobs {
		close(tblob)
	}
	for _, tblob := range bb.freelist {
		close(tblob)
	}

	stmt := bb.conn.Prep("DELETE FROM BlobBuffer WHERE rowid = $rowid;")
	del := func(tblob tblob) {
		stmt.Reset()
		stmt.SetInt64("$rowid", tblob.rowid)
		if _, err := stmt.Step(); err != nil && bb.err == nil {
			bb.err = err
		}
	}

	for _, tblob := range bb.blobs {
		del(tblob)
	}
	for _, tblob := range bb.freelist {
		del(tblob)
	}
	bb.blobs = nil
	bb.freelist = nil

	return bb.err
}
