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
	LOCALE_NOUSEROVERRIDE        = 0x80000000
	LOCALE_USE_CP_ACP            = 0x40000000
	LOCALE_RETURN_NUMBER         = 0x20000000
	LOCALE_RETURN_GENITIVE_NAMES = 0x10000000
	LOCALE_ALLOW_NEUTRAL_NAMES   = 0x08000000
)

var (
	procLocaleNameToLCID = modkernel32.NewProc("LocaleNameToLCID")
)

func LocaleNameToLCID(name *uint16, flags uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procLocaleNameToLCID.Addr(),
		2,
		uintptr(unsafe.Pointer(name)),
		uintptr(flags),
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
