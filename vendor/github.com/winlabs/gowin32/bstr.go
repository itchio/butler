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

func BstrToString(bstr *uint16) string {
	if bstr == nil {
		return ""
	}
	len := wrappers.SysStringLen(bstr)
	buf := make([]uint16, len)
	wrappers.RtlMoveMemory(
		(*byte)(unsafe.Pointer(&buf[0])),
		(*byte)(unsafe.Pointer(bstr)),
		uintptr(2*len))
	return syscall.UTF16ToString(buf)
}

func LpstrToString(lpstr *uint16) string {
	if lpstr == nil {
		return ""
	}
	len := wrappers.Lstrlen(lpstr)
	if len == 0 {
		return ""
	}
	buf := make([]uint16, len)
	wrappers.RtlMoveMemory(
		(*byte)(unsafe.Pointer(&buf[0])),
		(*byte)(unsafe.Pointer(lpstr)),
		uintptr(2*len))
	return syscall.UTF16ToString(buf)
}

func MakeDoubleNullTerminatedLpstr(items ...string) *uint16 {
	chars := []uint16{}
	for _, s := range items {
		chars = append(chars, syscall.StringToUTF16(s)...)
	}
	chars = append(chars, 0)
	return &chars[0]
}
