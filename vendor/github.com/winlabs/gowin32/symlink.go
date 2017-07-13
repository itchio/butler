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

	"syscall"
	"unsafe"
)

type SymbolicLinkData struct {
	SubstituteName string
	PrintName      string
	Relative       bool
}

func MakeSymbolicLink(symlinkPath string, targetPath string, isDirectory bool) error {
	var flags uint32
	if isDirectory {
		flags |= wrappers.SYMBOLIC_LINK_FLAG_DIRECTORY
	}
	err := wrappers.CreateSymbolicLink(
		syscall.StringToUTF16Ptr(symlinkPath),
		syscall.StringToUTF16Ptr(targetPath),
		flags)
	if err != nil {
		return NewWindowsError("CreateSymbolicLink", err)
	}
	return nil
}

func GetSymbolicLink(symlinkPath string) (*SymbolicLinkData, error) {
	file, err := wrappers.CreateFile(
		syscall.StringToUTF16Ptr(symlinkPath),
		wrappers.FILE_READ_EA,
		wrappers.FILE_SHARE_READ|wrappers.FILE_SHARE_WRITE|wrappers.FILE_SHARE_DELETE,
		nil,
		wrappers.OPEN_EXISTING,
		wrappers.FILE_FLAG_OPEN_REPARSE_POINT|wrappers.FILE_FLAG_BACKUP_SEMANTICS,
		0)
	if err != nil {
		return nil, NewWindowsError("CreateFile", err)
	}
	defer wrappers.CloseHandle(file)
	buf := make([]byte, wrappers.MAXIMUM_REPARSE_DATA_BUFFER_SIZE)
	var bytesReturned uint32
	err = wrappers.DeviceIoControl(
		file,
		wrappers.FSCTL_GET_REPARSE_POINT,
		nil,
		0,
		&buf[0],
		wrappers.MAXIMUM_REPARSE_DATA_BUFFER_SIZE,
		&bytesReturned,
		nil)
	if err != nil {
		return nil, NewWindowsError("DeviceIoControl", err)
	}
	data := (*wrappers.REPARSE_DATA_BUFFER)(unsafe.Pointer(&buf[0]))
	if data.ReparseTag != wrappers.IO_REPARSE_TAG_SYMLINK {
		return nil, nil
	}
	substituteNameBuf := make([]uint16, data.SubstituteNameLength/2)
	printNameBuf := make([]uint16, data.PrintNameLength/2)
	wrappers.RtlMoveMemory(
		(*byte)(unsafe.Pointer(&substituteNameBuf[0])),
		&buf[unsafe.Sizeof(*data)+uintptr(data.SubstituteNameOffset)],
		uintptr(data.SubstituteNameLength))
	wrappers.RtlMoveMemory(
		(*byte)(unsafe.Pointer(&printNameBuf[0])),
		&buf[unsafe.Sizeof(*data)+uintptr(data.PrintNameOffset)],
		uintptr(data.PrintNameLength))
	return &SymbolicLinkData{
		SubstituteName: syscall.UTF16ToString(substituteNameBuf),
		PrintName:      syscall.UTF16ToString(printNameBuf),
		Relative:       (data.Flags & wrappers.SYMLINK_FLAG_RELATIVE) != 0,
	}, nil
}
