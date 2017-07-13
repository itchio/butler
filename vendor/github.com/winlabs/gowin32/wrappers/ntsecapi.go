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
	POLICY_VIEW_LOCAL_INFORMATION   = 0x0001
	POLICY_VIEW_AUDIT_INFORMATION   = 0x0002
	POLICY_GET_PRIVATE_INFORMATION  = 0x0004
	POLICY_TRUST_ADMIN              = 0x0008
	POLICY_CREATE_ACCOUNT           = 0x0010
	POLICY_CREATE_SECRET            = 0x0020
	POLICY_CREATE_PRIVILEGE         = 0x0040
	POLICY_SET_DEFAULT_QUOTA_LIMITS = 0x0080
	POLICY_SET_AUDIT_REQUIREMENTS   = 0x0100
	POLICY_AUDIT_LOG_ADMIN          = 0x0200
	POLICY_SERVER_ADMIN             = 0x0400
	POLICY_LOOKUP_NAMES             = 0x0800
	POLICY_ALL_ACCESS               = STANDARD_RIGHTS_REQUIRED | POLICY_VIEW_LOCAL_INFORMATION | POLICY_VIEW_AUDIT_INFORMATION | POLICY_GET_PRIVATE_INFORMATION | POLICY_TRUST_ADMIN | POLICY_CREATE_ACCOUNT | POLICY_CREATE_SECRET | POLICY_CREATE_PRIVILEGE | POLICY_SET_DEFAULT_QUOTA_LIMITS | POLICY_SET_AUDIT_REQUIREMENTS | POLICY_AUDIT_LOG_ADMIN | POLICY_SERVER_ADMIN | POLICY_LOOKUP_NAMES
)

const (
	SE_INTERACTIVE_LOGON_NAME             = "SeInteractiveLogonRight"
	SE_NETWORK_LOGON_NAME                 = "SeNetworkLogonRight"
	SE_BATCH_LOGON_NAME                   = "SeBatchLogonRight"
	SE_SERVICE_LOGON_NAME                 = "SeServiceLogonRight"
	SE_DENY_INTERACTIVE_LOGON_NAME        = "SeDenyInteractiveLogonRight"
	SE_DENY_NETWORK_LOGON_NAME            = "SeDenyNetworkLogonRight"
	SE_DENY_BATCH_LOGON_NAME              = "SeDenyBatchLogonRight"
	SE_DENY_SERVICE_LOGON_NAME            = "SeDenyServiceLogonRight"
	SE_REMOTE_INTERACTIVE_LOGON_NAME      = "SeRemoteInteractiveLogonRight"
	SE_DENY_REMOTE_INTERACTIVE_LOGON_NAME = "SeDenyRemoteInteractiveLogonRight"
)

var (
	procLsaAddAccountRights       = modadvapi32.NewProc("LsaAddAccountRights")
	procLsaClose                  = modadvapi32.NewProc("LsaClose")
	procLsaEnumerateAccountRights = modadvapi32.NewProc("LsaEnumerateAccountRights")
	procLsaFreeMemory             = modadvapi32.NewProc("LsaFreeMemory")
	procLsaNtStatusToWinError     = modadvapi32.NewProc("LsaNtStatusToWinError")
	procLsaOpenPolicy             = modadvapi32.NewProc("LsaOpenPolicy")
	procLsaRemoveAccountRights    = modadvapi32.NewProc("LsaRemoveAccountRights")
)

func LsaAddAccountRights(policyHandle syscall.Handle, accountSid *SID, userRights *UNICODE_STRING, countOfRights uint32) uint32 {
	r1, _, _ := syscall.Syscall6(
		procLsaAddAccountRights.Addr(),
		4,
		uintptr(policyHandle),
		uintptr(unsafe.Pointer(accountSid)),
		uintptr(unsafe.Pointer(userRights)),
		uintptr(countOfRights),
		0,
		0)
	return uint32(r1)
}

func LsaClose(objectHandle syscall.Handle) uint32 {
	r1, _, _ := syscall.Syscall(procLsaClose.Addr(), 1, uintptr(objectHandle), 0, 0)
	return uint32(r1)
}

func LsaEnumerateAccountRights(policyHandle syscall.Handle, accountSid *SID, userRights **UNICODE_STRING, countOfRights *uint32) uint32 {
	r1, _, _ := syscall.Syscall6(
		procLsaEnumerateAccountRights.Addr(),
		4,
		uintptr(policyHandle),
		uintptr(unsafe.Pointer(accountSid)),
		uintptr(unsafe.Pointer(userRights)),
		uintptr(unsafe.Pointer(countOfRights)),
		0,
		0)
	return uint32(r1)
}

func LsaFreeMemory(buffer *byte) uint32 {
	r1, _, _ := syscall.Syscall(procLsaFreeMemory.Addr(), 1, uintptr(unsafe.Pointer(buffer)), 0, 0)
	return uint32(r1)
}

func LsaNtStatusToWinError(status uint32) error {
	r1, _, _ := syscall.Syscall(procLsaNtStatusToWinError.Addr(), 1, uintptr(status), 0, 0)
	err := syscall.Errno(r1)
	if err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func LsaOpenPolicy(systemName *UNICODE_STRING, objectAttributes *OBJECT_ATTRIBUTES, desiredAccess uint32, policyHandle *syscall.Handle) uint32 {
	r1, _, _ := syscall.Syscall6(
		procLsaOpenPolicy.Addr(),
		4,
		uintptr(unsafe.Pointer(systemName)),
		uintptr(unsafe.Pointer(objectAttributes)),
		uintptr(desiredAccess),
		uintptr(unsafe.Pointer(policyHandle)),
		0,
		0)
	return uint32(r1)
}

func LsaRemoveAccountRights(policyHandle syscall.Handle, accountSid *SID, allRights bool, userRights *UNICODE_STRING, countOfRights uint32) uint32 {
	var allRightsRaw uint8
	if allRights {
		allRightsRaw = 1
	} else {
		allRightsRaw = 0
	}
	r1, _, _ := syscall.Syscall6(
		procLsaRemoveAccountRights.Addr(),
		5,
		uintptr(policyHandle),
		uintptr(unsafe.Pointer(accountSid)),
		uintptr(allRightsRaw),
		uintptr(unsafe.Pointer(userRights)),
		uintptr(countOfRights),
		0)
	return uint32(r1)
}
