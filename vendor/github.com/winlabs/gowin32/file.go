/*
 * Copyright (c) 2014-2016 MongoDB, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the license is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gowin32

import (
	"github.com/winlabs/gowin32/wrappers"

	"os"
	"syscall"
)

type FileShareMode uint32

const (
	FileShareExclusive FileShareMode = 0
	FileShareRead      FileShareMode = wrappers.FILE_SHARE_READ
	FileShareWrite     FileShareMode = wrappers.FILE_SHARE_WRITE
	FileShareDelete    FileShareMode = wrappers.FILE_SHARE_DELETE
)

type FileCreationDisposition uint32

const (
	FileCreateNew        FileCreationDisposition = wrappers.CREATE_NEW
	FileCreateAlways     FileCreationDisposition = wrappers.CREATE_ALWAYS
	FileOpenExisting     FileCreationDisposition = wrappers.OPEN_EXISTING
	FileOpenAlways       FileCreationDisposition = wrappers.OPEN_ALWAYS
	FileTruncateExisting FileCreationDisposition = wrappers.TRUNCATE_EXISTING
)

type FileAttributes uint32

const (
	FileAttributeReadOnly          FileAttributes = wrappers.FILE_ATTRIBUTE_READONLY
	FileAttributeHidden            FileAttributes = wrappers.FILE_ATTRIBUTE_HIDDEN
	FileAttributeSystem            FileAttributes = wrappers.FILE_ATTRIBUTE_SYSTEM
	FileAttributeDirectory         FileAttributes = wrappers.FILE_ATTRIBUTE_DIRECTORY
	FileAttributeArchive           FileAttributes = wrappers.FILE_ATTRIBUTE_ARCHIVE
	FileAttributeDevice            FileAttributes = wrappers.FILE_ATTRIBUTE_DEVICE
	FileAttributeNormal            FileAttributes = wrappers.FILE_ATTRIBUTE_NORMAL
	FileAttributeTemporary         FileAttributes = wrappers.FILE_ATTRIBUTE_TEMPORARY
	FileAttributeSparseFile        FileAttributes = wrappers.FILE_ATTRIBUTE_SPARSE_FILE
	FileAttributeReparsePoint      FileAttributes = wrappers.FILE_ATTRIBUTE_REPARSE_POINT
	FileAttributeCompressed        FileAttributes = wrappers.FILE_ATTRIBUTE_COMPRESSED
	FileAttributeOffline           FileAttributes = wrappers.FILE_ATTRIBUTE_OFFLINE
	FileAttributeNotContentIndexed FileAttributes = wrappers.FILE_ATTRIBUTE_NOT_CONTENT_INDEXED
	FileAttributeEncrypted         FileAttributes = wrappers.FILE_ATTRIBUTE_ENCRYPTED
	FileAttributeVirtual           FileAttributes = wrappers.FILE_ATTRIBUTE_VIRTUAL
)

type FileFlags uint32

const (
	FileFlagWriteThrough      FileFlags = wrappers.FILE_FLAG_WRITE_THROUGH
	FileFlagOverlapped        FileFlags = wrappers.FILE_FLAG_OVERLAPPED
	FileFlagNoBuffering       FileFlags = wrappers.FILE_FLAG_NO_BUFFERING
	FileFlagRandomAccess      FileFlags = wrappers.FILE_FLAG_RANDOM_ACCESS
	FileFlagSequentialScan    FileFlags = wrappers.FILE_FLAG_SEQUENTIAL_SCAN
	FileFlagDeleteOnClose     FileFlags = wrappers.FILE_FLAG_DELETE_ON_CLOSE
	FileFlagBackupSemantics   FileFlags = wrappers.FILE_FLAG_BACKUP_SEMANTICS
	FileFlagPOSIXSemantics    FileFlags = wrappers.FILE_FLAG_POSIX_SEMANTICS
	FileFlagOpenReparsePoint  FileFlags = wrappers.FILE_FLAG_OPEN_REPARSE_POINT
	FileFlagOpenNoRecall      FileFlags = wrappers.FILE_FLAG_OPEN_NO_RECALL
	FileFlagFirstPipeInstance FileFlags = wrappers.FILE_FLAG_FIRST_PIPE_INSTANCE
)

func OpenWindowsFile(fileName string, readWrite bool, shareMode FileShareMode, creationDisposition FileCreationDisposition, attributes FileAttributes, flags FileFlags) (*os.File, error) {
	var accessMask uint32 = wrappers.GENERIC_READ
	if readWrite {
		accessMask |= wrappers.GENERIC_WRITE
	}
	file, err := wrappers.CreateFile(
		syscall.StringToUTF16Ptr(fileName),
		accessMask,
		uint32(shareMode),
		nil,
		uint32(creationDisposition),
		uint32(attributes)|uint32(flags),
		0)
	if err != nil {
		return nil, NewWindowsError("CreateFile", err)
	}
	return os.NewFile(uintptr(file), fileName), nil
}

func ReadFileContents(fileName string) (string, error) {
	file, err := wrappers.CreateFile(
		syscall.StringToUTF16Ptr(fileName),
		wrappers.GENERIC_READ,
		wrappers.FILE_SHARE_READ|wrappers.FILE_SHARE_WRITE|wrappers.FILE_SHARE_DELETE,
		nil,
		wrappers.OPEN_EXISTING,
		0,
		0)
	if err != nil {
		return "", NewWindowsError("CreateFile", err)
	}
	defer wrappers.CloseHandle(file)
	size, err := wrappers.GetFileSize(file, nil)
	if err != nil {
		return "", NewWindowsError("GetFileSize", err)
	}
	if size == 0 {
		return "", nil
	}
	buf := make([]byte, size)
	var bytesRead uint32
	if err := wrappers.ReadFile(file, &buf[0], size, &bytesRead, nil); err != nil {
		return "", NewWindowsError("ReadFile", err)
	}
	return string(buf[0:bytesRead]), nil
}

func TouchFile(f *os.File) error {
	var now wrappers.FILETIME
	wrappers.GetSystemTimeAsFileTime(&now)
	if err := wrappers.SetFileTime(syscall.Handle(f.Fd()), nil, &now, &now); err != nil {
		return NewWindowsError("SetFileTime", err)
	}
	return nil
}
