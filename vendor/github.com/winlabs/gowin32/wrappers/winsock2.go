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
)

const (
	WSA_IO_PENDING        = ERROR_IO_PENDING
	WSA_IO_INCOMPLETE     = ERROR_IO_INCOMPLETE
	WSA_INVALID_HANDLE    = ERROR_INVALID_HANDLE
	WSA_INVALID_PARAMETER = ERROR_INVALID_PARAMETER
	WSA_NOT_ENOUGH_MEMORY = ERROR_NOT_ENOUGH_MEMORY
	WSA_OPERATION_ABORTED = ERROR_OPERATION_ABORTED
)

var (
	modws2_32 = syscall.NewLazyDLL("ws2_32.dll")

	procWSAGetLastError = modws2_32.NewProc("WSAGetLastError")
	procWSASetLastError = modws2_32.NewProc("WSASetLastError")
	procntohs           = modws2_32.NewProc("ntohs")
)

func WSAGetLastError() error {
	r1, _, _ := syscall.Syscall(procWSAGetLastError.Addr(), 0, 0, 0, 0)
	return syscall.Errno(r1)
}

func WSASetLastError(error syscall.Errno) {
	syscall.Syscall(procWSASetLastError.Addr(), 1, uintptr(error), 0, 0)
}

func Ntohs(netshort uint16) uint16 {
	r1, _, _ := syscall.Syscall(procntohs.Addr(), 1, uintptr(netshort), 0, 0)
	return uint16(r1)
}
