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
	ADS_CHASE_REFERRALS_NEVER       = 0x00
	ADS_CHASE_REFERRALS_SUBORDINATE = 0x20
	ADS_CHASE_REFERRALS_EXTERNAL    = 0x40
	ADS_CHASE_REFERRALS_ALWAYS      = ADS_CHASE_REFERRALS_SUBORDINATE | ADS_CHASE_REFERRALS_EXTERNAL
)

const (
	ADS_NAME_TYPE_1779                    = 1
	ADS_NAME_TYPE_CANONICAL               = 2
	ADS_NAME_TYPE_NT4                     = 3
	ADS_NAME_TYPE_DISPLAY                 = 4
	ADS_NAME_TYPE_DOMAIN_SIMPLE           = 5
	ADS_NAME_TYPE_ENTERPRISE_SIMPLE       = 6
	ADS_NAME_TYPE_GUID                    = 7
	ADS_NAME_TYPE_UNKNOWN                 = 8
	ADS_NAME_TYPE_USER_PRINCIPAL_NAME     = 9
	ADS_NAME_TYPE_CANONICAL_EX            = 10
	ADS_NAME_TYPE_SERVICE_PRINCIPAL_NAME  = 11
	ADS_NAME_TYPE_SID_OR_SID_HISTORY_NAME = 12
)

const (
	ADS_NAME_INITTYPE_DOMAIN = 1
	ADS_NAME_INITTYPE_SERVER = 2
	ADS_NAME_INITTYPE_GC     = 3
)

var (
	IID_IADsNameTranslate   = GUID{0xB1B272A3, 0x3625, 0x11D1, [8]byte{0xA3, 0xA4, 0x00, 0xC0, 0x4F, 0xB9, 0x50, 0xDC}}
	IID_IADsADSystemInfo    = GUID{0x5BB11929, 0xAFD1, 0x11D2, [8]byte{0x9C, 0xB9, 0x00, 0x00, 0xF8, 0x7A, 0x36, 0x9E}}
	IID_IADsWinNTSystemInfo = GUID{0x6C6D65DC, 0xAFD1, 0x11D2, [8]byte{0x9C, 0xB9, 0x00, 0x00, 0xF8, 0x7A, 0x36, 0x9E}}
	CLSID_NameTranslate     = GUID{0x274FAE1F, 0x3626, 0x11D1, [8]byte{0xA3, 0xA4, 0x00, 0xC0, 0x4F, 0xB9, 0x50, 0xDC}}
	CLSID_ADSystemInfo      = GUID{0x50B6327F, 0xAFD1, 0x11D2, [8]byte{0x9C, 0xB9, 0x00, 0x00, 0xF8, 0x7A, 0x36, 0x9E}}
	CLSID_WinNTSystemInfo   = GUID{0x66182EC4, 0xAFD1, 0x11D2, [8]byte{0x9C, 0xB9, 0x00, 0x00, 0xF8, 0x7A, 0x36, 0x9E}}
)

type IADsNameTranslateVtbl struct {
	IDispatchVtbl
	Put_ChaseReferral uintptr
	Init              uintptr
	InitEx            uintptr
	Set               uintptr
	Get               uintptr
	SetEx             uintptr
	GetEx             uintptr
}

type IADsNameTranslate struct {
	IDispatch
}

func (self *IADsNameTranslate) Put_ChaseReferral(chaseReferral int32) uint32 {
	vtbl := (*IADsNameTranslateVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_ChaseReferral,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(chaseReferral),
		0)
	return uint32(r1)
}

func (self *IADsNameTranslate) Init(setType int32, adsPath *uint16) uint32 {
	vtbl := (*IADsNameTranslateVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Init,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(setType),
		uintptr(unsafe.Pointer(adsPath)))
	return uint32(r1)
}

func (self *IADsNameTranslate) InitEx(setType int32, adsPath *uint16, userID *uint16, domain *uint16, password *uint16) uint32 {
	vtbl := (*IADsNameTranslateVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall6(
		vtbl.InitEx,
		6,
		uintptr(unsafe.Pointer(self)),
		uintptr(setType),
		uintptr(unsafe.Pointer(adsPath)),
		uintptr(unsafe.Pointer(userID)),
		uintptr(unsafe.Pointer(domain)),
		uintptr(unsafe.Pointer(password)))
	return uint32(r1)
}

func (self *IADsNameTranslate) Set(setType int32, adsPath *uint16) uint32 {
	vtbl := (*IADsNameTranslateVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Set,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(setType),
		uintptr(unsafe.Pointer(adsPath)))
	return uint32(r1)
}

func (self *IADsNameTranslate) Get(formatType int32, adsPath **uint16) uint32 {
	vtbl := (*IADsNameTranslateVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(formatType),
		uintptr(unsafe.Pointer(adsPath)))
	return uint32(r1)
}

type IADsADSystemInfoVtbl struct {
	IDispatchVtbl
	Get_UserName        uintptr
	Get_ComputerName    uintptr
	Get_SiteName        uintptr
	Get_DomainShortName uintptr
	Get_DomainDNSName   uintptr
	Get_ForestDNSName   uintptr
	Get_PDCRoleOwner    uintptr
	Get_SchemaRoleOwner uintptr
	Get_IsNativeMode    uintptr
	GetAnyDCName        uintptr
	GetDCSiteName       uintptr
	RefreshSchemaCache  uintptr
	GetTrees            uintptr
}

type IADsADSystemInfo struct {
	IDispatch
}

func (self *IADsADSystemInfo) Get_UserName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_UserName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_ComputerName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_ComputerName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_SiteName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_SiteName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_DomainShortName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_DomainShortName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_DomainDNSName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_DomainDNSName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_ForestDNSName(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_ForestDNSName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_PDCRoleOwner(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_PDCRoleOwner,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_SchemaRoleOwner(retval **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_SchemaRoleOwner,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) Get_IsNativeMode(retval *bool) uint32 {
	if retval == nil {
		return E_POINTER
	}
	retvalRaw := int16(VARIANT_FALSE)
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_IsNativeMode,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(&retvalRaw)),
		0)
	*retval = (retvalRaw != VARIANT_FALSE)
	return uint32(r1)
}

func (self *IADsADSystemInfo) GetAnyDCName(dcName **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.GetAnyDCName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(dcName)),
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) GetDCSiteName(server *uint16, siteName **uint16) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.GetDCSiteName,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(server)),
		uintptr(unsafe.Pointer(siteName)))
	return uint32(r1)
}

func (self *IADsADSystemInfo) RefreshSchemaCache() uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.RefreshSchemaCache,
		1,
		uintptr(unsafe.Pointer(self)),
		0,
		0)
	return uint32(r1)
}

func (self *IADsADSystemInfo) GetTrees(trees *VARIANT) uint32 {
	vtbl := (*IADsADSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.GetTrees,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(trees)),
		0)
	return uint32(r1)
}

type IADsWinNTSystemInfoVtbl struct {
	IDispatchVtbl
	Get_UserName     uintptr
	Get_ComputerName uintptr
	Get_DomainName   uintptr
	Get_PDC          uintptr
}

type IADsWinNTSystemInfo struct {
	IDispatch
}

func (self *IADsWinNTSystemInfo) Get_UserName(retval **uint16) uint32 {
	vtbl := (*IADsWinNTSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_UserName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsWinNTSystemInfo) Get_ComputerName(retval **uint16) uint32 {
	vtbl := (*IADsWinNTSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_ComputerName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsWinNTSystemInfo) Get_DomainName(retval **uint16) uint32 {
	vtbl := (*IADsWinNTSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_DomainName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}

func (self *IADsWinNTSystemInfo) Get_PDC(retval **uint16) uint32 {
	vtbl := (*IADsWinNTSystemInfoVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_PDC,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(retval)),
		0)
	return uint32(r1)
}
