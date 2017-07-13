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

type Facility uint16

const (
	FacilityNull     Facility = wrappers.FACILITY_NULL
	FacilityRPC      Facility = wrappers.FACILITY_RPC
	FacilityDispatch Facility = wrappers.FACILITY_DISPATCH
	FacilityStorage  Facility = wrappers.FACILITY_STORAGE
	FacilityITF      Facility = wrappers.FACILITY_ITF
	FacilityWin32    Facility = wrappers.FACILITY_WIN32
	FacilityWindows  Facility = wrappers.FACILITY_WINDOWS
)

type COMError uint32

const (
	COMErrorUnexpected      COMError = wrappers.E_UNEXPECTED
	COMErrorNotImplemented  COMError = wrappers.E_NOTIMPL
	COMErrorOutOfMemory     COMError = wrappers.E_OUTOFMEMORY
	COMErrorInvalidArgument COMError = wrappers.E_INVALIDARG
	COMErrorNoInterface     COMError = wrappers.E_NOINTERFACE
	COMErrorPointer         COMError = wrappers.E_POINTER
	COMErrorHandle          COMError = wrappers.E_HANDLE
	COMErrorAbort           COMError = wrappers.E_ABORT
	COMErrorFail            COMError = wrappers.E_FAIL
	COMErrorAccessDenied    COMError = wrappers.E_ACCESSDENIED
	COMErrorPending         COMError = wrappers.E_PENDING
)

var (
	COMErrorNoneMapped           = COMError(wrappers.HRESULT_FROM_WIN32(wrappers.ERROR_NONE_MAPPED))
	COMErrorCantAccessDomainInfo = COMError(wrappers.HRESULT_FROM_WIN32(wrappers.ERROR_CANT_ACCESS_DOMAIN_INFO))
	COMErrorNoSuchDomain         = COMError(wrappers.HRESULT_FROM_WIN32(wrappers.ERROR_NO_SUCH_DOMAIN))
)

func (self COMError) Error() string {
	var message *uint16
	_, err := wrappers.FormatMessage(
		wrappers.FORMAT_MESSAGE_ALLOCATE_BUFFER | wrappers.FORMAT_MESSAGE_IGNORE_INSERTS | wrappers.FORMAT_MESSAGE_FROM_SYSTEM,
		0,
		uint32(self),
		0,
		(*uint16)(unsafe.Pointer(&message)),
		65536,
		nil)
	if err != nil {
		return fmt.Sprintf("com error 0x%08X", uint32(self))
	}
	defer wrappers.LocalFree(syscall.Handle(unsafe.Pointer(message)))
	return strings.TrimRight(LpstrToString(message), "\r\n")
}

func (self COMError) GetFacility() Facility {
	return Facility(wrappers.HRESULT_FACILITY(uint32(self)))
}

func (self COMError) GetCode() uint16 {
	return wrappers.HRESULT_CODE(uint32(self))
}
