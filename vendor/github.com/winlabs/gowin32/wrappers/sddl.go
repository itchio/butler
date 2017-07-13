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
	procConvertSidToStringSidW = modadvapi32.NewProc("ConvertSidToStringSidW")
)

func ConvertSidToStringSid(sid *SID, stringSid **uint16) error {
	r1, _, e1 := syscall.Syscall(
		procConvertSidToStringSidW.Addr(),
		2,
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(stringSid)),
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
