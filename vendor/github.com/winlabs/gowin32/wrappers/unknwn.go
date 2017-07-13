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
	IID_IUnknown = GUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

type IUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

type IUnknown struct {
	Vtbl *IUnknownVtbl
}

func (self *IUnknown) QueryInterface(iid *GUID, object *uintptr) uint32 {
	r1, _, _ := syscall.Syscall(
		self.Vtbl.QueryInterface,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(object)))
	return uint32(r1)
}

func (self *IUnknown) AddRef() uint32 {
	r1, _, _ := syscall.Syscall(self.Vtbl.AddRef, 1, uintptr(unsafe.Pointer(self)), 0, 0)
	return uint32(r1)
}

func (self *IUnknown) Release() uint32 {
	r1, _, _ := syscall.Syscall(self.Vtbl.Release, 1, uintptr(unsafe.Pointer(self)), 0, 0)
	return uint32(r1)
}
