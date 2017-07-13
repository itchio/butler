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

type VARIANT struct {
	Vt        uint16
	Reserved1 uint16
	Reserved2 uint16
	Reserved3 uint16
	Val       [variantDataBytes/8]uint64
}

var (
	IID_IDispatch    = GUID{0x0020400, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	IID_IEnumVARIANT = GUID{0x0020404, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

type IDispatchVtbl struct {
	IUnknownVtbl
	GetTypeInfoCount uintptr
	GetTypeInfo      uintptr
	GetIDsOfNames    uintptr
	Invoke           uintptr
}

type IDispatch struct {
	IUnknown
}

type IEnumVARIANTVtbl struct {
	IUnknownVtbl
	Next  uintptr
	Skip  uintptr
	Reset uintptr
	Clone uintptr
}

type IEnumVARIANT struct {
	IUnknown
}

func (self *IEnumVARIANT) Next(celt uint32, rgVar *VARIANT, celtFetched *uint32) uint32 {
	vtbl := (*IEnumVARIANTVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall6(
		vtbl.Next,
		4,
		uintptr(unsafe.Pointer(self)),
		uintptr(celt),
		uintptr(unsafe.Pointer(rgVar)),
		uintptr(unsafe.Pointer(celtFetched)),
		0,
		0)
	return uint32(r1)
}

func (self *IEnumVARIANT) Skip(celt uint32) uint32 {
	vtbl := (*IEnumVARIANTVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Skip,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(celt),
		0)
	return uint32(r1)
}

func (self *IEnumVARIANT) Reset() uint32 {
	vtbl := (*IEnumVARIANTVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Reset,
		1,
		uintptr(unsafe.Pointer(self)),
		0,
		0)
	return uint32(r1)
}

func (self *IEnumVARIANT) Clone(ppEnum *IEnumVARIANT) uint32 {
	vtbl := (*IEnumVARIANTVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Clone,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(ppEnum)),
		0)
	return uint32(r1)
}
