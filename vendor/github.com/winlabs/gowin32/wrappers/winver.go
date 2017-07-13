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

package wrappers

import (
	"syscall"
	"unsafe"
)

var (
	modversion = syscall.NewLazyDLL("version.dll")

	procGetFileVersionInfoSizeW = modversion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = modversion.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = modversion.NewProc("VerQueryValueW")
)

func GetFileVersionInfo(filename *uint16, handle uint32, len uint32, data *byte) error {
	r1, _, e1 := syscall.Syscall6(
		procGetFileVersionInfoW.Addr(),
		4,
		uintptr(unsafe.Pointer(filename)),
		uintptr(handle),
		uintptr(len),
		uintptr(unsafe.Pointer(data)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetFileVersionInfoSize(filename *uint16, handle *uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetFileVersionInfoSizeW.Addr(),
		2,
		uintptr(unsafe.Pointer(filename)),
		uintptr(unsafe.Pointer(handle)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func VerQueryValue(block *byte, subBlock *uint16, buffer **byte, len *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procVerQueryValueW.Addr(),
		4,
		uintptr(unsafe.Pointer(block)),
		uintptr(unsafe.Pointer(subBlock)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(len)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}
