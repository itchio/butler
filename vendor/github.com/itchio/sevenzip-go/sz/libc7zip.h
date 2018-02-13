
#ifndef LIBC7ZIP_H
#define LIBC7ZIP_H

#ifdef _MSC_VER
#define MYEXPORT   __declspec( dllexport )
#else
#define MYEXPORT 
#endif

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif // __cplusplus

struct lib;
typedef struct lib lib;
MYEXPORT lib *lib_new();
MYEXPORT int32_t lib_get_last_error(lib *l);
MYEXPORT void lib_free(lib *l);

struct in_stream;
typedef struct in_stream in_stream;

struct out_stream;
typedef struct out_stream out_stream;

// InStream functions
typedef int (*read_cb_t)(int64_t id, void *data, int64_t size, int64_t *processed_size);
typedef int (*seek_cb_t)(int64_t id, int64_t offset, int32_t whence, int64_t *new_position);

// SequentialOutStream functions
typedef int (*write_cb_t)(int64_t id, const void *data, int64_t size, int64_t *processed_size);

// ExtractCallback functions
typedef void (*set_total_cb_t)(int64_t id, int64_t size);
typedef void (*set_completed_cb_t)(int64_t id, int64_t complete_value);
typedef out_stream *(*get_stream_cb_t)(int64_t id, int64_t index);
typedef void (*set_operation_result_cb_t)(int64_t id, int32_t operation_result);

typedef struct in_stream_def {
  int64_t id;
	seek_cb_t seek_cb;
	read_cb_t read_cb;
  char *ext;
  int64_t size;
} in_stream_def;

typedef struct out_stream_def {
  int64_t id;
  write_cb_t write_cb;
} out_stream_def;

MYEXPORT in_stream *in_stream_new();
MYEXPORT in_stream_def *in_stream_get_def(in_stream *is);
MYEXPORT void in_stream_commit_def(in_stream *is);
MYEXPORT void in_stream_free(in_stream *is);

MYEXPORT out_stream *out_stream_new();
MYEXPORT out_stream_def *out_stream_get_def(out_stream *s);
MYEXPORT void out_stream_free(out_stream *s);

struct archive;
typedef struct archive archive;
MYEXPORT archive *archive_open(lib *l, in_stream *is, int32_t by_signature);
MYEXPORT void archive_close(archive *a);
MYEXPORT void archive_free(archive *a);
MYEXPORT int64_t archive_get_item_count(archive *a);
MYEXPORT char *archive_get_archive_format(archive *a);

// copied from lib7zip.h so we don't have to include it
enum property_index {
  PROP_INDEX_BEGIN,

  kpidPackSize = PROP_INDEX_BEGIN, //(Packed Size)
  kpidAttrib, //(Attributes)
  kpidCTime, //(Created)
  kpidATime, //(Accessed)
  kpidMTime, //(Modified)
  kpidSolid, //(Solid)
  kpidEncrypted, //(Encrypted)
  kpidUser, //(User)
  kpidGroup, //(Group)
  kpidComment, //(Comment)
  kpidPhySize, //(Physical Size)
  kpidHeadersSize, //(Headers Size)
  kpidChecksum, //(Checksum)
  kpidCharacts, //(Characteristics)
  kpidCreatorApp, //(Creator Application)
  kpidTotalSize, //(Total Size)
  kpidFreeSpace, //(Free Space)
  kpidClusterSize, //(Cluster Size)
  kpidVolumeName, //(Label)
  kpidPath, //(FullPath)
  kpidIsDir, //(IsDir)
  kpidSize, //(Uncompressed Size)
  kpidSymLink, //(Symbolic link destination)
  kpidPosixAttrib, //(POSIX Attributes)

  PROP_INDEX_END
};

// copied from lib7zip.h so we don't have to include it
enum error_code
{
  LIB7ZIP_ErrorCode_Begin,

  LIB7ZIP_NO_ERROR = LIB7ZIP_ErrorCode_Begin,
  LIB7ZIP_UNKNOWN_ERROR,
  LIB7ZIP_NOT_INITIALIZE,
  LIB7ZIP_NEED_PASSWORD,
  LIB7ZIP_NOT_SUPPORTED_ARCHIVE,

  LIB7ZIP_ErrorCode_End
};

struct item;
typedef struct item item;
MYEXPORT item *archive_get_item(archive *a, int64_t index);
MYEXPORT int32_t item_get_archive_index(item *i);
MYEXPORT char *item_get_string_property(item *i, int32_t property_index, int32_t *success);
MYEXPORT void string_free(char *s);
MYEXPORT uint64_t item_get_uint64_property(item *i, int32_t property_index, int32_t *success);
MYEXPORT int32_t item_get_bool_property(item *i, int32_t property_index, int32_t *success);
MYEXPORT void item_free(item *i);
MYEXPORT int archive_extract_item(archive *a, item *i, out_stream *os);

struct extract_callback;
typedef struct extract_callback extract_callback;

typedef struct extract_callback_def {
  int64_t id;
  set_total_cb_t set_total_cb;
  set_completed_cb_t set_completed_cb;
  get_stream_cb_t get_stream_cb;
  set_operation_result_cb_t set_operation_result_cb;
} extract_callback_def;

MYEXPORT extract_callback *extract_callback_new();
MYEXPORT extract_callback_def *extract_callback_get_def(extract_callback *ec);
MYEXPORT void extract_callback_free(extract_callback *ec);

MYEXPORT int archive_extract_several(archive *a, int64_t *indices, int32_t num_indices, extract_callback *ec);

#ifdef __cplusplus
} // extern "C"
#endif // __cplusplus

#endif // LIBC7ZIP_H