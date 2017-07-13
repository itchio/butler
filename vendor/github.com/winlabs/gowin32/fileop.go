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
)

func Copy(oldFileName string, newFileName string, overwrite bool) error {
	err := wrappers.CopyFile(
		syscall.StringToUTF16Ptr(oldFileName),
		syscall.StringToUTF16Ptr(newFileName),
		!overwrite)
	if err != nil {
		return NewWindowsError("CopyFile", err)
	}
	return nil
}

func Delete(fileName string) error {
	if err := wrappers.DeleteFile(syscall.StringToUTF16Ptr(fileName)); err != nil {
		return NewWindowsError("DeleteFile", err)
	}
	return nil
}

func FileExists(fileName string) (bool, error) {
	var wfd wrappers.WIN32_FIND_DATA
	handle, err := wrappers.FindFirstFile(syscall.StringToUTF16Ptr(fileName), &wfd)
	if err == wrappers.ERROR_FILE_NOT_FOUND {
		return false, nil
	} else if err != nil {
		return false, NewWindowsError("FindFirstFile", err)
	}
	wrappers.FindClose(handle)
	return true, nil
}

func GetAttributes(fileName string) (FileAttributes, error) {
	attributes, err := wrappers.GetFileAttributes(syscall.StringToUTF16Ptr(fileName))
	if err != nil {
		return 0, NewWindowsError("GetFileAttributes", err)
	}
	return FileAttributes(attributes), err
}

func GetCompressedSize(fileName string) (uint64, error) {
	var fileSizeHigh uint32
	fileSizeLow, err := wrappers.GetCompressedFileSize(syscall.StringToUTF16Ptr(fileName), &fileSizeHigh)
	if err != nil {
		return 0, NewWindowsError("GetCompressedFileSize", err)
	}
	return (uint64(fileSizeHigh) << 32) | uint64(fileSizeLow), nil
}

func GetVolumePath(fileName string) (string, error) {
	buf := make([]uint16, wrappers.MAX_PATH)
	if err := wrappers.GetVolumePathName(syscall.StringToUTF16Ptr(fileName), &buf[0], wrappers.MAX_PATH); err != nil {
		return "", NewWindowsError("GetVolumePathName", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func Move(oldFileName string, newFileName string, overwrite bool) error {
	if overwrite {
		err := wrappers.MoveFileEx(
			syscall.StringToUTF16Ptr(oldFileName),
			syscall.StringToUTF16Ptr(newFileName),
			wrappers.MOVEFILE_REPLACE_EXISTING)
		if err != nil {
			return NewWindowsError("MoveFileEx", err)
		}
	} else {
		err := wrappers.MoveFile(
			syscall.StringToUTF16Ptr(oldFileName),
			syscall.StringToUTF16Ptr(newFileName))
		if err != nil {
			return NewWindowsError("MoveFile", err)
		}
	}
	return nil
}

func SetAttributes(fileName string, attributes FileAttributes) error {
	err := wrappers.SetFileAttributes(syscall.StringToUTF16Ptr(fileName), uint32(attributes))
	if err != nil {
		return NewWindowsError("SetFileAttributes", err)
	}
	return nil
}
