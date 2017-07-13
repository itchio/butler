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

	"syscall"
	"unsafe"
)

type AccountRightName string

const (
	AccountRightInteractiveLogon           AccountRightName = wrappers.SE_INTERACTIVE_LOGON_NAME
	AccountRightNetworkLogon               AccountRightName = wrappers.SE_NETWORK_LOGON_NAME
	AccountRightBatchLogon                 AccountRightName = wrappers.SE_BATCH_LOGON_NAME
	AccountRightServiceLogon               AccountRightName = wrappers.SE_SERVICE_LOGON_NAME
	AccountRightDenyInteractiveLogon       AccountRightName = wrappers.SE_DENY_INTERACTIVE_LOGON_NAME
	AccountRightDenyNetworkLogon           AccountRightName = wrappers.SE_DENY_NETWORK_LOGON_NAME
	AccountRightDenyBatchLogon             AccountRightName = wrappers.SE_DENY_BATCH_LOGON_NAME
	AccountRightDenyServiceLogon           AccountRightName = wrappers.SE_DENY_SERVICE_LOGON_NAME
	AccountRightRemoteInteractiveLogon     AccountRightName = wrappers.SE_REMOTE_INTERACTIVE_LOGON_NAME
	AccountRightDenyRemoteInteractiveLogon AccountRightName = wrappers.SE_DENY_REMOTE_INTERACTIVE_LOGON_NAME
)

type SecurityPolicy struct {
	handle syscall.Handle
}

func OpenLocalSecurityPolicy() (*SecurityPolicy, error) {
	var handle syscall.Handle
	status := wrappers.LsaOpenPolicy(
		nil,
		&wrappers.OBJECT_ATTRIBUTES{},
		wrappers.POLICY_ALL_ACCESS,
		&handle)
	if err := wrappers.LsaNtStatusToWinError(status); err != nil {
		return nil, err
	}
	return &SecurityPolicy{handle: handle}, nil
}

func (self *SecurityPolicy) Close() error {
	if self.handle != 0 {
		status := wrappers.LsaClose(self.handle)
		if err := wrappers.LsaNtStatusToWinError(status); err != nil {
			return err
		}
		self.handle = 0
	}
	return nil
}

func (self *SecurityPolicy) GetAccountRights(sid SecurityID) ([]AccountRightName, error) {
	var rights *wrappers.UNICODE_STRING
	var count uint32
	status := wrappers.LsaEnumerateAccountRights(self.handle, sid.sid, &rights, &count)
	if err := wrappers.LsaNtStatusToWinError(status); err != nil {
		return nil, err
	}
	defer wrappers.LsaFreeMemory((*byte)(unsafe.Pointer(rights)))
	rightNames := make([]AccountRightName, count)
	for i := uint32(0); i < count; i++ {
		buf := make([]uint16, rights.Length)
		wrappers.RtlMoveMemory(
			(*byte)(unsafe.Pointer(&buf[0])),
			(*byte)(unsafe.Pointer(rights.Buffer)),
			uintptr(rights.Length))
		rightNames[i] = AccountRightName(syscall.UTF16ToString(buf))
		rights = (*wrappers.UNICODE_STRING)(unsafe.Pointer(uintptr(unsafe.Pointer(rights)) + unsafe.Sizeof(*rights)))
	}
	return rightNames, nil
}

func (self *SecurityPolicy) AddAccountRight(sid SecurityID, right AccountRightName) error {
	var rightString wrappers.UNICODE_STRING
	wrappers.RtlInitUnicodeString(&rightString, syscall.StringToUTF16Ptr(string(right)))
	status := wrappers.LsaAddAccountRights(self.handle, sid.sid, &rightString, 1)
	if err := wrappers.LsaNtStatusToWinError(status); err != nil {
		return err
	}
	return nil
}

func (self *SecurityPolicy) RemoveAccountRight(sid SecurityID, right AccountRightName) error {
	var rightString wrappers.UNICODE_STRING
	wrappers.RtlInitUnicodeString(&rightString, syscall.StringToUTF16Ptr(string(right)))
	status := wrappers.LsaRemoveAccountRights(self.handle, sid.sid, false, &rightString, 1)
	if err := wrappers.LsaNtStatusToWinError(status); err != nil {
		return err
	}
	return nil
}

func (self *SecurityPolicy) RemoveAllAccountRights(sid SecurityID) error {
	status := wrappers.LsaRemoveAccountRights(self.handle, sid.sid, true, nil, 0)
	if err := wrappers.LsaNtStatusToWinError(status); err != nil {
		return err
	}
	return nil
}
