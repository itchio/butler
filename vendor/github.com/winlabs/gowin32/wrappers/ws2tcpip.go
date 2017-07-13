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
	procInetNtopW = modws2_32.NewProc("InetNtopW")
)

func InetNtop(family int32, addr *byte, stringBuf *uint16, stringBufSize uintptr) (*uint16, error) {
	WSASetLastError(0)
	r1, _, _ := syscall.Syscall6(
		procInetNtopW.Addr(),
		4,
		uintptr(family),
		uintptr(unsafe.Pointer(addr)),
		uintptr(unsafe.Pointer(stringBuf)),
		stringBufSize,
		0,
		0)
	if r1 == 0 {
		return nil, WSAGetLastError()
	}
	return (*uint16)(unsafe.Pointer(r1)), nil
}
