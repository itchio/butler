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
)

type ADSPNOperation int32

const (
	ADSPNOperationAdd     ADSPNOperation = wrappers.DS_SPN_ADD_SPN_OP
	ADSPNOperationReplace ADSPNOperation = wrappers.DS_SPN_REPLACE_SPN_OP
	ADSPNOperationDelete  ADSPNOperation = wrappers.DS_SPN_DELETE_SPN_OP
)

func RegisterADServerSPN(operation ADSPNOperation, serviceClass string, userObjectDN string) error {
	var userObjectDNRaw *uint16
	if userObjectDN != "" {
		userObjectDNRaw = syscall.StringToUTF16Ptr(userObjectDN)
	}
	err := wrappers.DsServerRegisterSpn(
		int32(operation),
		syscall.StringToUTF16Ptr(serviceClass),
		userObjectDNRaw)
	if err != nil {
		return NewWindowsError("DsServerRegisterSpn", err)
	}
	return nil
}
