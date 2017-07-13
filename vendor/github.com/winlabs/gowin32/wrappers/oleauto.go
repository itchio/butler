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

const (
	VARIANT_NOVALUEPROP    = 0x0001
	VARIANT_ALPHABOOL      = 0x0002
	VARIANT_NOUSEROVERRIDE = 0x0004
	VARIANT_LOCALBOOL      = 0x0010
)

var (
	modoleaut32 = syscall.NewLazyDLL("oleaut32.dll")

	procSysAllocString    = modoleaut32.NewProc("SysAllocString")
	procSysFreeString     = modoleaut32.NewProc("SysFreeString")
	procSysStringLen      = modoleaut32.NewProc("SysStringLen")
	procVariantChangeType = modoleaut32.NewProc("VariantChangeType")
	procVariantClear      = modoleaut32.NewProc("VariantClear")
	procVariantInit       = modoleaut32.NewProc("VariantInit")
)

func SysAllocString(psz *uint16) *uint16 {
	r1, _, _ := syscall.Syscall(procSysAllocString.Addr(), 1, uintptr(unsafe.Pointer(psz)), 0, 0)
	return (*uint16)(unsafe.Pointer(r1))
}

func SysFreeString(bstrString *uint16) {
	syscall.Syscall(procSysFreeString.Addr(), 1, uintptr(unsafe.Pointer(bstrString)), 0, 0)
}

func SysStringLen(bstr *uint16) uint32 {
	r1, _, _ := syscall.Syscall(procSysStringLen.Addr(), 1, uintptr(unsafe.Pointer(bstr)), 0, 0)
	return uint32(r1)
}

func VariantChangeType(dest *VARIANT, src *VARIANT, flags uint16, vt uint16) uint32 {
	r1, _, _ := syscall.Syscall6(
		procVariantChangeType.Addr(),
		4,
		uintptr(unsafe.Pointer(dest)),
		uintptr(unsafe.Pointer(src)),
		uintptr(flags),
		uintptr(vt),
		0,
		0)
	return uint32(r1)
}

func VariantClear(variant *VARIANT) uint32 {
	r1, _, _ := syscall.Syscall(procVariantClear.Addr(), 1, uintptr(unsafe.Pointer(variant)), 0, 0)
	return uint32(r1)
}

func VariantInit(variant *VARIANT) {
	syscall.Syscall(procVariantInit.Addr(), 1, uintptr(unsafe.Pointer(variant)), 0, 0)
}
