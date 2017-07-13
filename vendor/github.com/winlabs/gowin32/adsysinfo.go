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

	"unsafe"
)

type ADSystemInfo struct {
	object *wrappers.IADsADSystemInfo
}

func NewADSystemInfo() (*ADSystemInfo, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_ADSystemInfo,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_IADsADSystemInfo,
		&object)
	if wrappers.FAILED(hr) {
		return nil, NewWindowsError("CoCreateInstance", COMError(hr))
	}
	return &ADSystemInfo{object: (*wrappers.IADsADSystemInfo)(unsafe.Pointer(object))}, nil
}

func (self *ADSystemInfo) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *ADSystemInfo) GetUserName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsADSystemInfo::get_UserName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_UserName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsADSystemInfo::get_UserName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

func (self *ADSystemInfo) GetComputerName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsADSystemInfo::get_ComputerName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_ComputerName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsADSystemInfo::get_ComputerName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

func (self *ADSystemInfo) GetDomainShortName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsADSystemInfo::get_DomainShortName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_DomainShortName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsADSystemInfo::get_DomainShortName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

func (self *ADSystemInfo) GetDomainDNSName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsADSystemInfo::get_DomainDNSName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_DomainDNSName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsADSystemInfo::get_DomainDNSName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

type ADWinNTSystemInfo struct {
	object *wrappers.IADsWinNTSystemInfo
}

func NewADWinNTSystemInfo() (*ADWinNTSystemInfo, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_WinNTSystemInfo,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_IADsWinNTSystemInfo,
		&object)
	if wrappers.FAILED(hr) {
		return nil, NewWindowsError("CoCreateInstance", COMError(hr))
	}
	return &ADWinNTSystemInfo{object: (*wrappers.IADsWinNTSystemInfo)(unsafe.Pointer(object))}, nil
}

func (self *ADWinNTSystemInfo) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *ADWinNTSystemInfo) GetUserName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_UserName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_UserName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_UserName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

func (self *ADWinNTSystemInfo) GetComputerName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_ComputerName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_ComputerName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_ComputerName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}

func (self *ADWinNTSystemInfo) GetDomainName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_DomainName", COMErrorPointer)
	}
	var retvalRaw *uint16
	if hr := self.object.Get_DomainName(&retvalRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsWinNTSystemInfo::get_DomainName", COMError(hr))
	}
	return BstrToString(retvalRaw), nil
}
