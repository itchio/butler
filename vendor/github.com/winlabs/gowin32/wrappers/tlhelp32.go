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

const MAX_MODULE_NAME32 = 255

const (
	TH32CS_SNAPHEAPLIST = 0x00000001
	TH32CS_SNAPPROCESS  = 0x00000002
	TH32CS_SNAPTHREAD   = 0x00000004
	TH32CS_SNAPMODULE   = 0x00000008
	TH32CS_SNAPMODULE32 = 0x00000010
	TH32CS_SNAPALL      = TH32CS_SNAPHEAPLIST | TH32CS_SNAPPROCESS | TH32CS_SNAPTHREAD | TH32CS_SNAPMODULE
	TH32CS_INHERIT      = 0x80000000
)

type PROCESSENTRY32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [MAX_PATH]uint16
}

type MODULEENTRY32 struct {
	Size         uint32
	ModuleID     uint32
	ProcessID    uint32
	GlblcntUsage uint32
	ProccntUsage uint32
	ModBaseAddr  *byte
	ModBaseSize  uint32
	Module       syscall.Handle
	ModuleName   [MAX_MODULE_NAME32 + 1]uint16
	ExePath      [MAX_PATH]uint16
}

var (
	procCreateToolhelp32Snapshot = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procModule32FirstW           = modkernel32.NewProc("Module32FirstW")
	procModule32NextW            = modkernel32.NewProc("Module32NextW")
	procProcess32FirstW          = modkernel32.NewProc("Process32FirstW")
	procProcess32NextW           = modkernel32.NewProc("Process32NextW")
)

func CreateToolhelp32Snapshot(flags uint32, processID uint32) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procCreateToolhelp32Snapshot.Addr(),
		2,
		uintptr(flags),
		uintptr(processID),
		0)
	handle := syscall.Handle(r1)
	if handle == INVALID_HANDLE_VALUE {
		if e1 != ERROR_SUCCESS {
			return handle, e1
		} else {
			return handle, syscall.EINVAL
		}
	}
	return handle, nil
}

func Module32First(snapshot syscall.Handle, me *MODULEENTRY32) error {
	r1, _, e1 := syscall.Syscall(
		procModule32FirstW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(me)),
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

func Module32Next(snapshot syscall.Handle, me *MODULEENTRY32) error {
	r1, _, e1 := syscall.Syscall(
		procModule32NextW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(me)),
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

func Process32First(snapshot syscall.Handle, pe *PROCESSENTRY32) error {
	r1, _, e1 := syscall.Syscall(
		procProcess32FirstW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)),
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

func Process32Next(snapshot syscall.Handle, pe *PROCESSENTRY32) error {
	r1, _, e1 := syscall.Syscall(
		procProcess32NextW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pe)),
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
