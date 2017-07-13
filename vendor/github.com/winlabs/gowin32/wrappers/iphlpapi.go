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
	modiphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procGetTcpTable = modiphlpapi.NewProc("GetTcpTable")
)

func GetTcpTable(tcpTable *MIB_TCPTABLE, size *uint32, order bool) error {
	var orderRaw int32
	if order {
		orderRaw = 1
	} else {
		orderRaw = 0
	}
	r1, _, _ := syscall.Syscall(
		procGetTcpTable.Addr(),
		3,
		uintptr(unsafe.Pointer(tcpTable)),
		uintptr(unsafe.Pointer(size)),
		uintptr(orderRaw))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}
