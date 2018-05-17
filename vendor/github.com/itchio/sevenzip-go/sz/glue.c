
#define GLUE_IMPLEMENT 1
#include "glue.h"

#include "_cgo_export.h"

#include <stdio.h>

#ifdef __WIN32
#include <windows.h>
#define MY_DLOPEN LoadLibrary
#define MY_DLSYM GetProcAddress
#define MY_LIBHANDLE HMODULE
#else // __WIN32
#include <dlfcn.h>
#define MY_DLOPEN(x) dlopen((x), RTLD_LAZY|RTLD_LOCAL)
#define MY_DLSYM dlsym
#define MY_LIBHANDLE void*
#endif // !__WIN32

#define LOADSYM(sym) { \
  sym ## _ = (void*) MY_DLSYM(dynlib, #sym); \
  if (! sym ## _) { \
    fprintf(stderr, "Could not load symbol %s\n", #sym); \
    fflush(stderr); \
    return 1; \
  } \
}

MY_LIBHANDLE dynlib;

int libc7zip_initialize(char *lib_path) {
  dynlib = MY_DLOPEN(lib_path);
  if (!dynlib) {
    fprintf(stderr, "Could not load %s\n", lib_path);
    fflush(stderr);
    return 1;
  }

  LOADSYM(lib_new)
  LOADSYM(lib_get_last_error)
  LOADSYM(lib_get_version)
  LOADSYM(lib_free)

  LOADSYM(in_stream_new)
  LOADSYM(in_stream_get_def)
  LOADSYM(in_stream_commit_def)
  LOADSYM(in_stream_free)

  LOADSYM(out_stream_new)
  LOADSYM(out_stream_get_def)
  LOADSYM(out_stream_free)

  LOADSYM(archive_open)
  LOADSYM(archive_close)
  LOADSYM(archive_free)
  LOADSYM(archive_get_archive_format)
  LOADSYM(archive_get_item_count)
  LOADSYM(archive_get_item)

  LOADSYM(item_get_archive_index)
  LOADSYM(item_get_string_property)
  LOADSYM(string_free)
  LOADSYM(item_get_uint64_property)
  LOADSYM(item_get_bool_property)
  LOADSYM(item_free)

  LOADSYM(archive_extract_item)
  LOADSYM(archive_extract_several)

  LOADSYM(extract_callback_new)
  LOADSYM(extract_callback_get_def)
  LOADSYM(extract_callback_free)

  return 0;
}

lib *libc7zip_lib_new() {
  return lib_new_();
}

void libc7zip_lib_free(lib *l) {
  return lib_free_(l);
}

int32_t libc7zip_lib_get_last_error(lib *l) {
  return lib_get_last_error_(l);
}

char *libc7zip_lib_get_version(lib *l) {
  return lib_get_version_(l);
}

//-----------------

in_stream *libc7zip_in_stream_new() {
  return in_stream_new_();
}

in_stream_def *libc7zip_in_stream_get_def(in_stream *is) {
  return in_stream_get_def_(is);
}

void libc7zip_in_stream_commit_def(in_stream *is) {
  in_stream_commit_def_(is);
}

void libc7zip_in_stream_free(in_stream *is) {
  return in_stream_free_(is);
}

//-----------------

out_stream *libc7zip_out_stream_new() {
  return out_stream_new_();
}

out_stream_def *libc7zip_out_stream_get_def(out_stream *os) {
  return out_stream_get_def_(os);
}

void libc7zip_out_stream_free(out_stream *os) {
  return out_stream_free_(os);
}

//-----------------

archive *libc7zip_archive_open(lib *l, in_stream *is, int32_t by_signature) {
  return archive_open_(l, is, by_signature);
}

void libc7zip_archive_close(archive *a) {
  return archive_close_(a);
}

void libc7zip_archive_free(archive *a) {
  return archive_free_(a);
}

char *libc7zip_archive_get_archive_format(archive *a) {
  return archive_get_archive_format_(a);
}

int64_t libc7zip_archive_get_item_count(archive *a) {
  return archive_get_item_count_(a);
}

item *libc7zip_archive_get_item(archive *a, int64_t index) {
  return archive_get_item_(a, index);
}

int32_t libc7zip_item_get_archive_index(item *i) {
  return item_get_archive_index_(i);
}

char *libc7zip_item_get_string_property(item *i, int32_t property_index, int32_t *success) {
  return item_get_string_property_(i, property_index, success);
}

void libc7zip_string_free(char *s) {
  string_free_(s);
}

uint64_t libc7zip_item_get_uint64_property(item *i, int32_t property_index, int32_t *success) {
  return item_get_uint64_property_(i, property_index, success);
}

int32_t libc7zip_item_get_bool_property(item *i, int32_t property_index, int32_t *success) {
  return item_get_bool_property_(i, property_index, success);
}

void libc7zip_item_free(item *i) {
  return item_free_(i);
}

int libc7zip_archive_extract_item(archive *a, item *i, out_stream *os) {
  return archive_extract_item_(a, i, os);
}

int libc7zip_archive_extract_several(archive *a, int64_t *indices, int32_t num_indices, extract_callback *ec) {
  return archive_extract_several_(a, indices, num_indices, ec);
}

//-----------------

extract_callback *libc7zip_extract_callback_new() {
  return extract_callback_new_();
}

extract_callback_def *libc7zip_extract_callback_get_def(extract_callback *ec) {
  return extract_callback_get_def_(ec);
}

void libc7zip_extract_callback_free(extract_callback *ec) {
  extract_callback_free_(ec);
}

// Gateway functions

int inSeekGo_cgo(int64_t id, int64_t offset, int32_t whence, int64_t *new_position) {
  return inSeekGo(id, offset, whence, new_position);
}

int inReadGo_cgo(int64_t id, void *data, int64_t size, int64_t *processed_size) {
  return inReadGo(id, data, size, processed_size);
}

int outWriteGo_cgo(int64_t id, const void *data, int64_t size, int64_t *processed_size) {
  return outWriteGo(id, (void*) data, size, processed_size);
}

void ecSetTotalGo_cgo(int64_t id, int64_t size) {
  ecSetTotalGo(id, size);
}

void ecSetCompletedGo_cgo(int64_t id, int64_t size) {
  ecSetCompletedGo(id, size);
}

out_stream *ecGetStreamGo_cgo(int64_t id, int64_t index) {
  return ecGetStreamGo(id, index);
}

void ecSetOperationResultGo_cgo(int64_t id, int32_t result) {
  ecSetOperationResultGo(id, result);
}

