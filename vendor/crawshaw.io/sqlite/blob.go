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

package sqlite

// #include <blocking_step.h>
// #include <sqlite3.h>
// #include <stdlib.h>
// #include <stdint.h>
//
// int wrap_blob_read(sqlite3_blob* blob, uintptr_t p, int n, int off) {
//	return sqlite3_blob_read(blob, (void*)p, n, off);
// }
import "C"
import (
	"io"
	"runtime"
	"unsafe"
)

var cmain = C.CString("main")
var ctemp = C.CString("temp")

// OpenBlob opens a blob in a particular {database,table,column,row}.
//
// https://www.sqlite.org/c3ref/blob_open.html
func (conn *Conn) OpenBlob(dbn, table, column string, row int64, write bool) (*Blob, error) {
	var cdb *C.char
	switch {
	case dbn == "" || dbn == "main":
		cdb = cmain
	case dbn == "temp":
		cdb = ctemp
	default:
		cdb = C.CString(dbn)
		defer C.free(unsafe.Pointer(cdb))
	}
	var flags C.int
	if write {
		flags = 1
	}

	ctable := C.CString(table)
	ccolumn := C.CString(column)
	defer func() {
		C.free(unsafe.Pointer(ctable))
		C.free(unsafe.Pointer(ccolumn))
	}()

	blob := &Blob{conn: conn}

	for {
		conn.count++
		if err := conn.interrupted("Conn.OpenBlob", ""); err != nil {
			return nil, err
		}
		switch res := C.sqlite3_blob_open(conn.conn, cdb, ctable, ccolumn, C.sqlite3_int64(row), flags, &blob.blob); res {
		case C.SQLITE_LOCKED_SHAREDCACHE:
			if res := C.wait_for_unlock_notify(conn.conn, conn.unlockNote); res != C.SQLITE_OK {
				return nil, conn.reserr("Conn.OpenBlob(Wait)", "", res)
			}
			// loop
		case C.SQLITE_OK:
			blob.size = int64(C.sqlite3_blob_bytes(blob.blob))
			return blob, nil
		default:
			return nil, conn.extreserr("Conn.OpenBlob", "", res)
		}
	}
}

// Blob provides streaming access to SQLite blobs.
type Blob struct {
	io.ReadWriteSeeker
	io.ReaderAt
	io.WriterAt
	io.Closer

	conn *Conn
	blob *C.sqlite3_blob
	off  int64
	size int64
}

func (blob *Blob) readAt(p []byte, off int64) C.int {
	n := len(p)
	// This is incorrect. TODO: Remove when we can.
	//
	// We should pass unsafe.Pointer directly to sqlite3_blob_read,
	// this lets the Go runtime pin the memory correctly.
	// However with Go 1.10 we see a particularly nasty variant of
	// https://golang.org/issue/14210, so until its resolved, we
	// rely on the fact that this memory won't move anywhere yet.
	pp := uintptr(unsafe.Pointer(&p[0]))
	res := C.wrap_blob_read(blob.blob, C.uintptr_t(pp), C.int(n), C.int(off))
	runtime.KeepAlive(p)
	return res
}

// https://www.sqlite.org/c3ref/blob_read.html
func (blob *Blob) ReadAt(p []byte, off int64) (n int, err error) {
	if blob.blob == nil {
		return 0, errInvalidBlob
	}
	if err := blob.conn.interrupted("Blob.ReadAt", ""); err != nil {
		return 0, err
	}
	res := blob.readAt(p, off)
	if err := blob.conn.reserr("Blob.ReadAt", "", res); err != nil {
		return 0, err
	}
	return len(p), nil
}

// https://www.sqlite.org/c3ref/blob_write.html
func (blob *Blob) WriteAt(p []byte, off int64) (n int, err error) {
	if blob.blob == nil {
		return 0, errInvalidBlob
	}
	if err := blob.conn.interrupted("Blob.WriteAt", ""); err != nil {
		return 0, err
	}
	v := unsafe.Pointer(&p[0])
	res := C.sqlite3_blob_write(blob.blob, v, C.int(len(p)), C.int(off))
	if err := blob.conn.reserr("Blob.WriteAt", "", res); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (blob *Blob) Read(p []byte) (n int, err error) {
	if blob.off >= blob.size {
		return 0, io.EOF
	}
	if rem := blob.size - blob.off; int64(len(p)) > rem {
		p = p[:rem]
	}
	n, err = blob.ReadAt(p, blob.off)
	blob.off += int64(n)
	return n, err
}

func (blob *Blob) Write(p []byte) (n int, err error) {
	if rem := blob.size - blob.off; int64(len(p)) > rem {
		return 0, io.ErrShortWrite
	}
	n, err = blob.WriteAt(p, blob.off)
	blob.off += int64(n)
	return n, err
}

func (blob *Blob) Seek(offset int64, whence int) (int64, error) {
	const (
		SeekStart   = 0
		SeekCurrent = 1
		SeekEnd     = 2
	)
	switch whence {
	case SeekStart:
		// use offset directly
	case SeekCurrent:
		offset += blob.off
	case SeekEnd:
		offset += blob.size
	}
	if offset < 0 {
		var buf [20]byte
		return -1, Error{
			Code: SQLITE_ERROR,
			Loc:  "Blob.Seek",
			Msg:  "attempting to seek before beginning of blob: " + string(itoa(buf[:], offset)),
		}
	}
	blob.off = offset
	return offset, nil
}

// Size returns the total size of a blob.
func (blob *Blob) Size() int64 {
	return blob.size
}

// https://www.sqlite.org/c3ref/blob_close.html
func (blob *Blob) Close() error {
	if blob.blob == nil {
		return errInvalidBlob
	}
	err := blob.conn.reserr("Blob.Close", "", C.sqlite3_blob_close(blob.blob))
	blob.blob = nil
	return err
}

var errInvalidBlob = Error{Code: SQLITE_ERROR, Msg: "invalid blob"}

// TODO: Blob Reopen
