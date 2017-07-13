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

type UNICODE_STRING struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

type RTL_USER_PROCESS_PARAMETERS struct {
	Reserved1     [16]byte
	Reserved2     [10]uintptr
	ImagePathName UNICODE_STRING
	CommandLine   UNICODE_STRING
}

type PEB struct {
	Reserved1              [2]byte
	BeingDebugged          byte
	Reserved2              [1]byte
	Reserved3              [2]uintptr
	Ldr                    uintptr
	ProcessParameters      uintptr
	Reserved4              [104]byte
	Reserved5              [52]uintptr
	PostProcessInitRoutine uintptr
	Reserved6              [128]byte
	Reserved7              [1]uintptr
	SessionId              uint32
}

type OBJECT_ATTRIBUTES struct {
	Length                   uint32
	RootDirectory            syscall.Handle
	ObjectName               *UNICODE_STRING
	Attributes               uint32
	SecurityDescriptor       uintptr
	SecurityQualityOfService uintptr
}

type PROCESS_BASIC_INFORMATION struct {
	Reserved1       uintptr
	PebBaseAddress  uintptr
	Reserved2       [2]uintptr
	UniqueProcessId uintptr
	Reserved3       uintptr
}

const (
	ProcessBasicInformation = 0
	ProcessWow64Information = 26
)

func NT_SUCCESS(status uint32) bool {
	return int32(uintptr(status)) >= 0
}

func NT_INFORMATION(status uint32) bool {
	return (status >> 30) == 1
}

func NT_WARNING(status uint32) bool {
	return (status >> 30) == 2
}

func NT_ERROR(status uint32) bool {
	return (status >> 30) == 3
}

var (
	modntdll = syscall.NewLazyDLL("ntdll.dll")

	procNtQueryInformationProcess = modntdll.NewProc("NtQueryInformationProcess")
	procRtlFreeUnicodeString      = modntdll.NewProc("RtlFreeUnicodeString")
	procRtlInitUnicodeString      = modntdll.NewProc("RtlInitUnicodeString")
)

func NtQueryInformationProcess(processHandle syscall.Handle, processInformationClass int32, processInformation *byte, processInformationLength uint32, returnLength *uint32) uint32 {
	r1, _, _ := syscall.Syscall6(
		procNtQueryInformationProcess.Addr(),
		5,
		uintptr(processHandle),
		uintptr(processInformationClass),
		uintptr(unsafe.Pointer(processInformation)),
		uintptr(processInformationLength),
		uintptr(unsafe.Pointer(returnLength)),
		0)
	return uint32(r1)
}

func RtlFreeUnicodeString(unicodeString *UNICODE_STRING) {
	syscall.Syscall(
		procRtlFreeUnicodeString.Addr(),
		1,
		uintptr(unsafe.Pointer(unicodeString)),
		0,
		0)
}

func RtlInitUnicodeString(destinationString *UNICODE_STRING, sourceString *uint16) {
	syscall.Syscall(
		procRtlInitUnicodeString.Addr(),
		2,
		uintptr(unsafe.Pointer(destinationString)),
		uintptr(unsafe.Pointer(sourceString)),
		0)
}
