package sz

/*
#cgo !windows LDFLAGS: -ldl

#include <stdlib.h> // for C.free
#include "glue.h"

// forward declaration for gateway functions
int inReadGo_cgo(int64_t id, void *data, int64_t size, int64_t *processed_size);
int inSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);

int outWriteGo_cgo(int64_t id, const void *data, int64_t size, int64_t *processed_size);

void ecSetTotalGo_cgo(int64_t id, int64_t size);
void ecSetCompletedGo_cgo(int64_t id, int64_t size);
out_stream *ecGetStreamGo_cgo(int64_t id, int64_t index);
void ecSetOperationResultGo_cgo(int64_t id, int32_t result);
*/
import "C"
import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/pkg/errors"
)

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}

type InStream struct {
	reader ReaderAtCloser
	size   int64
	id     int64
	offset int64
	strm   *C.in_stream
	err    error

	Stats *ReadStats

	ChunkSize int64
}

type OutStream struct {
	writer io.WriteCloser
	id     int64
	strm   *C.out_stream
	closed bool
	err    error

	ChunkSize int64
}

type Lib struct {
	lib *C.lib
}

var lazyInitDone = false

func lazyInit() error {
	if lazyInitDone {
		return nil
	}

	libPath := "unsupported-os"
	switch runtime.GOOS {
	case "windows":
		libPath = "c7zip.dll"
	case "linux":
		libPath = "libc7zip.so"
	case "darwin":
		libPath = "libc7zip.dylib"
	}

	execPath, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}

	libPath = filepath.Join(filepath.Dir(execPath), libPath)

	cLibPath := C.CString(libPath)
	defer C.free(unsafe.Pointer(cLibPath))

	ret := C.libc7zip_initialize(cLibPath)
	if ret != 0 {
		return fmt.Errorf("could not initialize libc7zip")
	}

	lazyInitDone = true
	return nil
}

func NewLib() (*Lib, error) {
	err := lazyInit()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	lib := C.libc7zip_lib_new()
	if lib == nil {
		return nil, fmt.Errorf("could not create new lib")
	}

	l := &Lib{
		lib: lib,
	}
	return l, nil
}

func (l *Lib) Error() error {
	le := (ErrorCode)(C.libc7zip_lib_get_last_error(l.lib))
	switch le {
	case ErrCodeNoError:
		return nil
	case ErrCodeUnknownError:
		return ErrUnknownError
	case ErrCodeNotInitialize:
		return ErrNotInitialize
	case ErrCodeNeedPassword:
		return ErrNeedPassword
	case ErrCodeNotSupportedArchive:
		return ErrNotSupportedArchive
	default:
		return ErrUnknownError
	}
}

func (l *Lib) Free() {
	C.libc7zip_lib_free(l.lib)
}

func NewInStream(reader ReaderAtCloser, ext string, size int64) (*InStream, error) {
	strm := C.libc7zip_in_stream_new()
	if strm == nil {
		return nil, fmt.Errorf("could not create new InStream")
	}

	in := &InStream{
		reader: reader,
		size:   size,
		offset: 0,
		strm:   strm,
	}
	reserveInStreamId(in)

	def := C.libc7zip_in_stream_get_def(strm)
	def.size = C.int64_t(in.size)
	def.ext = C.CString(ext)
	def.id = C.int64_t(in.id)
	def.read_cb = (C.read_cb_t)(unsafe.Pointer(C.inReadGo_cgo))
	def.seek_cb = (C.seek_cb_t)(unsafe.Pointer(C.inSeekGo_cgo))

	C.libc7zip_in_stream_commit_def(strm)

	return in, nil
}

func (in *InStream) SetExt(ext string) {
	strm := in.strm
	if strm == nil {
		return
	}

	def := C.libc7zip_in_stream_get_def(strm)
	def.ext = C.CString(ext)
	C.libc7zip_in_stream_commit_def(strm)
}

func (in *InStream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		in.offset = offset
	case io.SeekCurrent:
		in.offset += offset
	case io.SeekEnd:
		in.offset = in.size + offset
	}

	return in.offset, nil
}

func (in *InStream) Free() {
	if in.id > 0 {
		freeInStreamId(in.id)
		in.id = 0
	}

	if in.strm != nil {
		C.libc7zip_in_stream_free(in.strm)
		in.strm = nil
	}
}

func (in *InStream) Error() error {
	return in.err
}

func NewOutStream(writer io.WriteCloser) (*OutStream, error) {
	strm := C.libc7zip_out_stream_new()
	if strm == nil {
		return nil, fmt.Errorf("could not create new OutStream")
	}

	out := &OutStream{
		writer: writer,
		strm:   strm,
	}
	reserveOutStreamId(out)

	def := C.libc7zip_out_stream_get_def(strm)
	def.id = C.int64_t(out.id)
	def.write_cb = (C.write_cb_t)(unsafe.Pointer(C.outWriteGo_cgo))

	return out, nil
}

func (out *OutStream) Close() error {
	if out.id > 0 {
		freeOutStreamId(out.id)
		out.id = 0
		return out.writer.Close()
	}

	// already closed
	return nil
}

func (out *OutStream) Free() {
	if out.strm != nil {
		C.libc7zip_out_stream_free(out.strm)
		out.strm = nil
	}
}

func (out *OutStream) Error() error {
	return out.err
}

type Archive struct {
	arch *C.archive
	in   *InStream
	lib  *Lib
}

func (lib *Lib) OpenArchive(in *InStream, bySignature bool) (*Archive, error) {
	cBySignature := C.int32_t(0)
	if bySignature {
		cBySignature = 1
	}

	arch := C.libc7zip_archive_open(lib.lib, in.strm, cBySignature)
	if arch == nil {
		err := coalesceErrors(in.Error(), lib.Error(), ErrUnknownError)
		return nil, errors.WithStack(err)
	}

	a := &Archive{
		arch: arch,
		in:   in,
		lib:  lib,
	}
	return a, nil
}

func (a *Archive) Close() {
	C.libc7zip_archive_close(a.arch)
}

func (a *Archive) Free() {
	C.libc7zip_archive_free(a.arch)
}

func (a *Archive) GetArchiveFormat() string {
	cstr := C.libc7zip_archive_get_archive_format(a.arch)
	if cstr == nil {
		return ""
	}

	defer C.libc7zip_string_free(cstr)
	return C.GoString(cstr)
}

func (a *Archive) GetItemCount() (int64, error) {
	res := int64(C.libc7zip_archive_get_item_count(a.arch))
	if res < 0 {
		err := coalesceErrors(a.in.Error(), a.lib.Error(), ErrUnknownError)
		return 0, errors.WithStack(err)
	}
	return res, nil
}

func coalesceErrors(errors ...error) error {
	for _, e := range errors {
		if e != nil {
			return e
		}
	}
	return nil
}

type Item struct {
	item *C.item
}

func (a *Archive) GetItem(index int64) *Item {
	item := C.libc7zip_archive_get_item(a.arch, C.int64_t(index))
	if item == nil {
		return nil
	}

	return &Item{
		item: item,
	}
}

type PropertyIndex int32

var (
	// Packed Size
	PidPackSize PropertyIndex = C.kpidPackSize
	// Attributes
	PidAttrib PropertyIndex = C.kpidAttrib
	// Created
	PidCTime PropertyIndex = C.kpidCTime
	// Accessed
	PidATime PropertyIndex = C.kpidATime
	// Modified
	PidMTime PropertyIndex = C.kpidMTime
	// Solid
	PidSolid PropertyIndex = C.kpidSolid
	// Encrypted
	PidEncrypted PropertyIndex = C.kpidEncrypted
	// User
	PidUser PropertyIndex = C.kpidUser
	// Group
	PidGroup PropertyIndex = C.kpidGroup
	// Comment
	PidComment PropertyIndex = C.kpidComment
	// Physical Size
	PidPhySize PropertyIndex = C.kpidPhySize
	// Headers Size
	PidHeadersSize PropertyIndex = C.kpidHeadersSize
	// Checksum
	PidChecksum PropertyIndex = C.kpidChecksum
	// Characteristics
	PidCharacts PropertyIndex = C.kpidCharacts
	// Creator Application
	PidCreatorApp PropertyIndex = C.kpidCreatorApp
	// Total Size
	PidTotalSize PropertyIndex = C.kpidTotalSize
	// Free Space
	PidFreeSpace PropertyIndex = C.kpidFreeSpace
	// Cluster Size
	PidClusterSize PropertyIndex = C.kpidClusterSize
	// Label
	PidVolumeName PropertyIndex = C.kpidVolumeName
	// FullPath
	PidPath PropertyIndex = C.kpidPath
	// IsDir
	PidIsDir PropertyIndex = C.kpidIsDir
	// Uncompressed Size
	PidSize PropertyIndex = C.kpidSize
	// Symbolic link destination
	PidSymLink PropertyIndex = C.kpidSymLink
	// POSIX attributes
	PidPosixAttrib PropertyIndex = C.kpidPosixAttrib
)

type ErrorCode int32

var (
	ErrCodeNoError ErrorCode = C.LIB7ZIP_NO_ERROR

	ErrCodeUnknownError ErrorCode = C.LIB7ZIP_UNKNOWN_ERROR
	ErrUnknownError               = errors.New("Unknown 7-zip error")

	ErrCodeNotInitialize ErrorCode = C.LIB7ZIP_NOT_INITIALIZE
	ErrNotInitialize               = errors.New("7-zip not initialized")

	ErrCodeNeedPassword ErrorCode = C.LIB7ZIP_NEED_PASSWORD
	ErrNeedPassword               = errors.New("Password required to extract archive with 7-zip")

	ErrCodeNotSupportedArchive ErrorCode = C.LIB7ZIP_NOT_SUPPORTED_ARCHIVE
	ErrNotSupportedArchive               = errors.New("Archive type not supported by 7-zip")
)

func (i *Item) GetArchiveIndex() int64 {
	return int64(C.libc7zip_item_get_archive_index(i.item))
}

func (i *Item) GetStringProperty(id PropertyIndex) (string, bool) {
	var success = C.int32_t(0)

	cstr := C.libc7zip_item_get_string_property(i.item, C.int32_t(id), &success)
	if cstr == nil {
		return "", false
	}

	defer C.libc7zip_string_free(cstr)
	return C.GoString(cstr), success == 1
}

func (i *Item) GetUInt64Property(id PropertyIndex) (uint64, bool) {
	var success = C.int32_t(0)
	val := uint64(C.libc7zip_item_get_uint64_property(i.item, C.int32_t(id), &success))
	return val, success == 1
}

func (i *Item) GetBoolProperty(id PropertyIndex) (bool, bool) {
	var success = C.int32_t(0)
	val := C.libc7zip_item_get_bool_property(i.item, C.int32_t(id), &success) != 0
	return val, success == 1
}

func (i *Item) Free() {
	C.libc7zip_item_free(i.item)
}

func (a *Archive) Extract(i *Item, out *OutStream) error {
	// returns a boolean, truthiness indicates success
	success := C.libc7zip_archive_extract_item(a.arch, i.item, out.strm)
	if success == 0 {
		err := coalesceErrors(a.in.Error(), out.Error(), a.lib.Error(), ErrUnknownError)
		return errors.WithStack(err)
	}

	return nil
}

type ExtractCallbackFuncs interface {
	SetProgress(completed int64, total int64)
	GetStream(item *Item) (*OutStream, error)
}

type ExtractCallback struct {
	id    int64
	cb    *C.extract_callback
	funcs ExtractCallbackFuncs

	total   int64
	archive *Archive
	item    *Item
	out     *OutStream
	errors  []error
}

func NewExtractCallback(funcs ExtractCallbackFuncs) (*ExtractCallback, error) {
	cb := C.libc7zip_extract_callback_new()
	if cb == nil {
		return nil, fmt.Errorf("could not create new ExtractCallback")
	}

	ec := &ExtractCallback{
		funcs: funcs,
		cb:    cb,
	}
	reserveExtractCallbackId(ec)

	def := C.libc7zip_extract_callback_get_def(cb)
	def.id = C.int64_t(ec.id)
	def.set_total_cb = (C.set_total_cb_t)(unsafe.Pointer(C.ecSetTotalGo_cgo))
	def.set_completed_cb = (C.set_completed_cb_t)(unsafe.Pointer(C.ecSetCompletedGo_cgo))
	def.get_stream_cb = (C.get_stream_cb_t)(unsafe.Pointer(C.ecGetStreamGo_cgo))
	def.set_operation_result_cb = (C.set_operation_result_cb_t)(unsafe.Pointer(C.ecSetOperationResultGo_cgo))

	return ec, nil
}

func (ec *ExtractCallback) Errors() []error {
	return ec.errors
}

func (ec *ExtractCallback) Free() {
	C.libc7zip_extract_callback_free(ec.cb)
}

func (a *Archive) ExtractSeveral(indices []int64, ec *ExtractCallback) error {
	ec.archive = a

	// returns a boolean, truthiness indicates success
	success := C.libc7zip_archive_extract_several(a.arch, (*C.int64_t)(unsafe.Pointer(&indices[0])), C.int32_t(len(indices)), ec.cb)
	ec.archive = nil
	if success == 0 {
		err := coalesceErrors(a.in.Error(), a.lib.Error(), ErrUnknownError)
		return errors.WithStack(err)
	}

	return nil
}

//export inSeekGo
func inSeekGo(id int64, offset int64, whence int32, newPosition unsafe.Pointer) int {
	p, ok := inStreams.Load(id)
	if !ok {
		return 1
	}
	in, ok := (p).(*InStream)
	if !ok {
		return 1
	}

	newOffset, err := in.Seek(offset, int(whence))
	if err != nil {
		in.err = err
		return 1
	}

	newPosPtr := (*int64)(newPosition)
	*newPosPtr = newOffset

	in.err = nil
	return 0
}

//export inReadGo
func inReadGo(id int64, data unsafe.Pointer, size int64, processedSize unsafe.Pointer) int {
	p, ok := inStreams.Load(id)
	if !ok {
		return 1
	}
	in, ok := (p).(*InStream)
	if !ok {
		return 1
	}

	if in.ChunkSize > 0 && size > in.ChunkSize {
		size = in.ChunkSize
	}

	if in.offset+size > in.size {
		size = in.size - in.offset
	}

	if in.Stats != nil {
		in.Stats.RecordRead(in.offset, size)
	}

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	readBytes, err := in.reader.ReadAt(buf, in.offset)
	if err != nil {
		in.err = err
		return 1
	}

	in.offset += int64(readBytes)

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(readBytes)

	in.err = nil
	return 0
}

//export outWriteGo
func outWriteGo(id int64, data unsafe.Pointer, size int64, processedSize unsafe.Pointer) int {
	p, ok := outStreams.Load(id)
	if !ok {
		return 1
	}
	out, ok := (p).(*OutStream)
	if !ok {
		return 1
	}

	if out.ChunkSize > 0 && size > out.ChunkSize {
		size = out.ChunkSize
	}

	h := reflect.SliceHeader{
		Data: uintptr(data),
		Cap:  int(size),
		Len:  int(size),
	}
	buf := *(*[]byte)(unsafe.Pointer(&h))

	writtenBytes, err := out.writer.Write(buf)
	if err != nil {
		out.err = err
		return 1
	}

	processedSizePtr := (*int64)(processedSize)
	*processedSizePtr = int64(writtenBytes)

	out.err = nil
	return 0
}

//export ecSetTotalGo
func ecSetTotalGo(id int64, size int64) {
	p, ok := extractCallbacks.Load(id)
	if !ok {
		return
	}
	ec, ok := (p).(*ExtractCallback)
	if !ok {
		return
	}

	ec.total = size
}

//export ecSetCompletedGo
func ecSetCompletedGo(id int64, completed int64) {
	p, ok := extractCallbacks.Load(id)
	if !ok {
		return
	}
	ec, ok := (p).(*ExtractCallback)
	if !ok {
		return
	}

	ec.funcs.SetProgress(completed, ec.total)
}

//export ecGetStreamGo
func ecGetStreamGo(id int64, index int64) *C.out_stream {
	p, ok := extractCallbacks.Load(id)
	if !ok {
		return nil
	}
	ec, ok := (p).(*ExtractCallback)
	if !ok {
		return nil
	}

	ec.item = ec.archive.GetItem(int64(index))
	if ec.item == nil {
		ec.errors = append(ec.errors, errors.Errorf("sz: no Item for index %d", index))
		return nil
	}

	out, err := ec.funcs.GetStream(ec.item)
	if err != nil {
		ec.errors = append(ec.errors, err)
		return nil
	}

	if out != nil {
		ec.out = out
		return out.strm
	}
	return nil
}

//export ecSetOperationResultGo
func ecSetOperationResultGo(id int64, result int32) {
	p, ok := extractCallbacks.Load(id)
	if !ok {
		return
	}
	ec, ok := (p).(*ExtractCallback)
	if !ok {
		return
	}

	if ec.item != nil {
		ec.item.Free()
		ec.item = nil
	}

	if ec.out != nil {
		err := ec.out.Close()
		if err != nil {
			ec.errors = append(ec.errors, errors.WithStack(err))
		}
		ec.out.Free()
		ec.out = nil
	}

	// TODO: so, if result isn't NArchive::NExtract::NOperationResult::kOK
	// then something went wrong with the extraction, should we call
	// GetLastError() and append it somewhere ?
	if result != 0 {
		err := coalesceErrors(ec.archive.lib.Error(), ErrUnknownError)
		ec.errors = append(ec.errors, errors.WithStack(err))
	}
}
