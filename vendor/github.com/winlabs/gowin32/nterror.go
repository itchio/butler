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

	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

type NTError uint32

func (self NTError) Error() string {
	hModule, err := wrappers.LoadLibrary(syscall.StringToUTF16Ptr("ntdll.dll"))
	if err != nil {
		return fmt.Sprintf("nt error 0x%08X", uint32(self))
	}
	defer wrappers.FreeLibrary(hModule)
	var message *uint16
	_, err = wrappers.FormatMessage(
		wrappers.FORMAT_MESSAGE_ALLOCATE_BUFFER | wrappers.FORMAT_MESSAGE_FROM_SYSTEM | wrappers.FORMAT_MESSAGE_FROM_HMODULE,
		uintptr(hModule),
		uint32(self),
		0,
		(*uint16)(unsafe.Pointer(&message)),
		0,
		nil)
	if err != nil {
		return fmt.Sprintf("nt error 0x%08X", uint32(self))
	}
	defer wrappers.LocalFree(syscall.Handle(unsafe.Pointer(message)))
	return strings.TrimRight(LpstrToString(message), "\r\n")
}
