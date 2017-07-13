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

type VerRelOp uint8

const (
	VerEqual        VerRelOp = wrappers.VER_EQUAL
	VerGreater      VerRelOp = wrappers.VER_GREATER
	VerGreaterEqual VerRelOp = wrappers.VER_GREATER_EQUAL
	VerLess         VerRelOp = wrappers.VER_LESS
	VerLessEqual    VerRelOp = wrappers.VER_LESS_EQUAL
)

type VerLogOp uint8

const (
	VerAnd VerLogOp = wrappers.VER_AND
	VerOr  VerLogOp = wrappers.VER_OR
)

type VerPlatform uint32

const (
	VerPlatformWin32s    VerPlatform = wrappers.VER_PLATFORM_WIN32s
	VerPlatformWindows9x VerPlatform = wrappers.VER_PLATFORM_WIN32_WINDOWS
	VerPlatformWindowsNT VerPlatform = wrappers.VER_PLATFORM_WIN32_NT
)

type VerSuite uint16

const (
	VerSuiteSmallBusiness           VerSuite = wrappers.VER_SUITE_SMALLBUSINESS
	VerSuiteEnterprise              VerSuite = wrappers.VER_SUITE_ENTERPRISE
	VerSuiteBackOffice              VerSuite = wrappers.VER_SUITE_BACKOFFICE
	VerSuiteCommunications          VerSuite = wrappers.VER_SUITE_COMMUNICATIONS
	VerSuiteTerminal                VerSuite = wrappers.VER_SUITE_TERMINAL
	VerSuiteSmallBusinessRestricted VerSuite = wrappers.VER_SUITE_SMALLBUSINESS_RESTRICTED
	VerSuiteEmbeddedNT              VerSuite = wrappers.VER_SUITE_EMBEDDEDNT
	VerSuiteDataCenter              VerSuite = wrappers.VER_SUITE_DATACENTER
	VerSuiteSingleUserTS            VerSuite = wrappers.VER_SUITE_SINGLEUSERTS
	VerSuitePersonal                VerSuite = wrappers.VER_SUITE_PERSONAL
	VerSuiteBlade                   VerSuite = wrappers.VER_SUITE_BLADE
	VerSuiteEmbeddedRestricted      VerSuite = wrappers.VER_SUITE_EMBEDDED_RESTRICTED
	VerSuiteSecurityAppliance       VerSuite = wrappers.VER_SUITE_SECURITY_APPLIANCE
	VerSuiteStorageServer           VerSuite = wrappers.VER_SUITE_STORAGE_SERVER
	VerSuiteComputeServer           VerSuite = wrappers.VER_SUITE_COMPUTE_SERVER
	VerSuiteWHServer                VerSuite = wrappers.VER_SUITE_WH_SERVER
)

type VerProductType uint8

const (
	VerProductWorkstation      VerProductType = wrappers.VER_NT_WORKSTATION
	VerProductDomainController VerProductType = wrappers.VER_NT_DOMAIN_CONTROLLER
	VerProductServer           VerProductType = wrappers.VER_NT_SERVER
)

type VersionCheck struct {
	osvi          wrappers.OSVERSIONINFOEX
	typeMask      uint32
	conditionMask uint64
}

func (self *VersionCheck) MajorVersion(op VerRelOp, value uint) {
	self.osvi.MajorVersion = uint32(value)
	self.typeMask |= wrappers.VER_MAJORVERSION
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_MAJORVERSION, uint8(op))
}

func (self *VersionCheck) MinorVersion(op VerRelOp, value uint) {
	self.osvi.MinorVersion = uint32(value)
	self.typeMask |= wrappers.VER_MINORVERSION
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_MINORVERSION, uint8(op))
}

func (self *VersionCheck) BuildNumber(op VerRelOp, value uint) {
	self.osvi.BuildNumber = uint32(value)
	self.typeMask |= wrappers.VER_BUILDNUMBER
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_BUILDNUMBER, uint8(op))
}

func (self *VersionCheck) Platform(op VerRelOp, value VerPlatform) {
	self.osvi.PlatformId = uint32(value)
	self.typeMask |= wrappers.VER_PLATFORMID
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_PLATFORMID, uint8(op))
}

func (self *VersionCheck) ServicePackMajor(op VerRelOp, value uint) {
	self.osvi.ServicePackMajor = uint16(value)
	self.typeMask |= wrappers.VER_SERVICEPACKMAJOR
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_SERVICEPACKMAJOR, uint8(op))
}

func (self *VersionCheck) ServicePackMinor(op VerRelOp, value uint) {
	self.osvi.ServicePackMinor = uint16(value)
	self.typeMask |= wrappers.VER_SERVICEPACKMINOR
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_SERVICEPACKMINOR, uint8(op))
}

func (self *VersionCheck) Suite(op VerLogOp, value VerSuite) {
	self.osvi.SuiteMask = uint16(value)
	self.typeMask |= wrappers.VER_SUITENAME
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_SUITENAME, uint8(op))
}

func (self *VersionCheck) ProductType(op VerRelOp, value VerProductType) {
	self.osvi.ProductType = uint8(value)
	self.typeMask |= wrappers.VER_PRODUCT_TYPE
	self.conditionMask = wrappers.VerSetConditionMask(self.conditionMask, wrappers.VER_PRODUCT_TYPE, uint8(op))
}

func (self *VersionCheck) Verify() (bool, error) {
	self.osvi.OSVersionInfoSize = uint32(unsafe.Sizeof(self.osvi))
	if err := wrappers.VerifyVersionInfo(&self.osvi, self.typeMask, self.conditionMask); err != nil {
		if err == wrappers.ERROR_OLD_WIN_VERSION {
			return false, nil
		}
		return false, NewWindowsError("VerifyVersionInfo", err)
	}
	return true, nil
}

type OSVersionInfo struct {
	MajorVersion     uint
	MinorVersion     uint
	BuildNumber      uint
	PlatformId       VerPlatform
	ServicePackName  string
	ServicePackMajor uint
	ServicePackMinor uint
	SuiteMask        VerSuite
	ProductType      VerProductType
}

func GetWindowsVersion() (*OSVersionInfo, error) {
	var osvi wrappers.OSVERSIONINFOEX
	osvi.OSVersionInfoSize = uint32(unsafe.Sizeof(osvi))
	if err := wrappers.GetVersionEx(&osvi); err != nil {
		return nil, NewWindowsError("GetVersionEx", err)
	}
	return &OSVersionInfo{
		MajorVersion:     uint(osvi.MajorVersion),
		MinorVersion:     uint(osvi.MinorVersion),
		BuildNumber:      uint(osvi.BuildNumber),
		PlatformId:       VerPlatform(osvi.PlatformId),
		ServicePackName:  syscall.UTF16ToString(osvi.CSDVersion[:]),
		ServicePackMajor: uint(osvi.ServicePackMajor),
		ServicePackMinor: uint(osvi.ServicePackMinor),
		SuiteMask:        VerSuite(osvi.SuiteMask),
		ProductType:      VerProductType(osvi.ProductType),
	}, nil
}
