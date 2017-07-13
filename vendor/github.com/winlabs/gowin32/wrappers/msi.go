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
	INSTALLMESSAGE_FATALEXIT      = 0x00000000
	INSTALLMESSAGE_ERROR          = 0x01000000
	INSTALLMESSAGE_WARNING        = 0x02000000
	INSTALLMESSAGE_USER           = 0x03000000
	INSTALLMESSAGE_INFO           = 0x04000000
	INSTALLMESSAGE_FILESINUSE     = 0x05000000
	INSTALLMESSAGE_RESOLVESOURCE  = 0x06000000
	INSTALLMESSAGE_OUTOFDISKSPACE = 0x07000000
	INSTALLMESSAGE_ACTIONSTART    = 0x08000000
	INSTALLMESSAGE_ACTIONDATA     = 0x09000000
	INSTALLMESSAGE_PROGRESS       = 0x0A000000
	INSTALLMESSAGE_COMMONDATA     = 0x0B000000
	INSTALLMESSAGE_INITIALIZE     = 0x0C000000
	INSTALLMESSAGE_TERMINATE      = 0x0D000000
	INSTALLMESSAGE_SHOWDIALOG     = 0x0E000000
	INSTALLMESSAGE_RMFILESINUSE   = 0x19000000
	INSTALLMESSAGE_INSTALLSTART   = 0x1A000000
	INSTALLMESSAGE_INSTALLEND     = 0x1B000000
)

const (
	INSTALLUILEVEL_NOCHANGE = 0
	INSTALLUILEVEL_DEFAULT  = 1
	INSTALLUILEVEL_NONE     = 2
	INSTALLUILEVEL_BASIC    = 3
	INSTALLUILEVEL_REDUCED  = 4
	INSTALLUILEVEL_FULL     = 5
	
	INSTALLUILEVEL_ENDDIALOG     = 0x0080
	INSTALLUILEVEL_PROGRESSONLY  = 0x0040
	INSTALLUILEVEL_HIDECANCEL    = 0x0020
	INSTALLUILEVEL_SOURCERESONLY = 0x0100
)

const (
	INSTALLSTATE_BADCONFIG    = -6
	INSTALLSTATE_INCOMPLETE   = -5
	INSTALLSTATE_SOURCEABSENT = -4
	INSTALLSTATE_MOREDATA     = -3
	INSTALLSTATE_INVALIDARG   = -2
	INSTALLSTATE_UNKNOWN      = -1
	INSTALLSTATE_BROKEN       = 0
	INSTALLSTATE_ADVERTISED   = 1
	INSTALLSTATE_ABSENT       = 2
	INSTALLSTATE_LOCAL        = 3
	INSTALLSTATE_SOURCE       = 4
	INSTALLSTATE_DEFAULT      = 5
)

const (
	INSTALLLEVEL_DEFAULT = 0
	INSTALLLEVEL_MINIMUM = 1
	INSTALLLEVEL_MAXIMUM = 0xFFFF
)

const (
	INSTALLLOGMODE_FATALEXIT      = 1 << (INSTALLMESSAGE_FATALEXIT >> 24)
	INSTALLLOGMODE_ERROR          = 1 << (INSTALLMESSAGE_ERROR >> 24)
	INSTALLLOGMODE_WARNING        = 1 << (INSTALLMESSAGE_WARNING >> 24)
	INSTALLLOGMODE_USER           = 1 << (INSTALLMESSAGE_USER >> 24)
	INSTALLLOGMODE_INFO           = 1 << (INSTALLMESSAGE_INFO >> 24)
	INSTALLLOGMODE_RESOLVESOURCE  = 1 << (INSTALLMESSAGE_RESOLVESOURCE >> 24)
	INSTALLLOGMODE_OUTOFDISKSPACE = 1 << (INSTALLMESSAGE_OUTOFDISKSPACE >> 24)
	INSTALLLOGMODE_ACTIONSTART    = 1 << (INSTALLMESSAGE_ACTIONSTART >> 24)
	INSTALLLOGMODE_ACTIONDATA     = 1 << (INSTALLMESSAGE_ACTIONDATA >> 24)
	INSTALLLOGMODE_COMMONDATA     = 1 << (INSTALLMESSAGE_COMMONDATA >> 24)
	INSTALLLOGMODE_PROPERTYDUMP   = 1 << (INSTALLMESSAGE_PROGRESS >> 24)
	INSTALLLOGMODE_VERBOSE        = 1 << (INSTALLMESSAGE_INITIALIZE >> 24)
	INSTALLLOGMODE_EXTRADEBUG     = 1 << (INSTALLMESSAGE_TERMINATE >> 24)
	INSTALLLOGMODE_LOGONLYONERROR = 1 << (INSTALLMESSAGE_SHOWDIALOG >> 24)
	INSTALLLOGMODE_PROGRESS       = 1 << (INSTALLMESSAGE_PROGRESS >> 24)
	INSTALLLOGMODE_INITIALIZE     = 1 << (INSTALLMESSAGE_INITIALIZE >> 24)
	INSTALLLOGMODE_TERMINATE      = 1 << (INSTALLMESSAGE_TERMINATE >> 24)
	INSTALLLOGMODE_SHOWDIALOG     = 1 << (INSTALLMESSAGE_SHOWDIALOG >> 24)
	INSTALLLOGMODE_FILESINUSE     = 1 << (INSTALLMESSAGE_FILESINUSE >> 24)
	INSTALLLOGMODE_RMFILESINUSE   = 1 << (INSTALLMESSAGE_RMFILESINUSE >> 24)
	INSTALLLOGMODE_INSTALLSTART   = 1 << (INSTALLMESSAGE_INSTALLSTART >> 24)
	INSTALLLOGMODE_INSTALLEND     = 1 << (INSTALLMESSAGE_INSTALLEND >> 24)
)

const (
	INSTALLLOGATTRIBUTES_APPEND        = 1 << 0
	INSTALLLOGATTRIBUTES_FLUSHEACHLINE = 1 << 1
)

const (
	INSTALLPROPERTY_PACKAGENAME          = "PackageName"
	INSTALLPROPERTY_TRANSFORMS           = "Transforms"
	INSTALLPROPERTY_LANGUAGE             = "Language"
	INSTALLPROPERTY_PRODUCTNAME          = "ProductName"
	INSTALLPROPERTY_ASSIGNMENTTYPE       = "AssignmentType"
	INSTALLPROPERTY_INSTANCETYPE         = "InstanceType"
	INSTALLPROPERTY_AUTHORIZED_LUA_APP   = "AuthorizedLUAApp"
	INSTALLPROPERTY_PACKAGECODE          = "PackageCode"
	INSTALLPROPERTY_VERSION              = "Version"
	INSTALLPROPERTY_PRODUCTICON          = "ProductIcon"
	INSTALLPROPERTY_INSTALLEDPRODUCTNAME = "InstalledProductName"
	INSTALLPROPERTY_VERSIONSTRING        = "VersionString"
	INSTALLPROPERTY_HELPLINK             = "HelpLink"
	INSTALLPROPERTY_HELPTELEPHONE        = "HelpTelephone"
	INSTALLPROPERTY_INSTALLLOCATION      = "InstallLocation"
	INSTALLPROPERTY_INSTALLSOURCE        = "InstallSource"
	INSTALLPROPERTY_INSTALLDATE          = "InstallDate"
	INSTALLPROPERTY_PUBLISHER            = "Publisher"
	INSTALLPROPERTY_LOCALPACKAGE         = "LocalPackage"
	INSTALLPROPERTY_URLINFOABOUT         = "URLInfoAbout"
	INSTALLPROPERTY_URLUPDATEINFO        = "URLUpdateInfo"
	INSTALLPROPERTY_VERSIONMINOR         = "VersionMinor"
	INSTALLPROPERTY_VERSIONMAJOR         = "VersionMajor"
	INSTALLPROPERTY_PRODUCTID            = "ProductID"
	INSTALLPROPERTY_REGCOMPANY           = "RegCompany"
	INSTALLPROPERTY_REGOWNER             = "RegOwner"
	INSTALLPROPERTY_INSTALLEDLANGUAGE    = "InstalledLanguage"
)

var (
	modmsi = syscall.NewLazyDLL("msi.dll")

	procMsiCloseHandle          = modmsi.NewProc("MsiCloseHandle")
	procMsiConfigureProductExW  = modmsi.NewProc("MsiConfigureProductExW")
	procMsiConfigureProductW    = modmsi.NewProc("MsiConfigureProductW")
	procMsiEnableLogW           = modmsi.NewProc("MsiEnableLogW")
	procMsiEnumRelatedProductsW = modmsi.NewProc("MsiEnumRelatedProductsW")
	procMsiGetComponentPathW    = modmsi.NewProc("MsiGetComponentPathW")
	procMsiGetProductInfoW      = modmsi.NewProc("MsiGetProductInfoW")
	procMsiGetProductPropertyW  = modmsi.NewProc("MsiGetProductPropertyW")
	procMsiGetPropertyW         = modmsi.NewProc("MsiGetPropertyW")
	procMsiInstallProductW      = modmsi.NewProc("MsiInstallProductW")
	procMsiOpenPackageW         = modmsi.NewProc("MsiOpenPackageW")
	procMsiOpenProductW         = modmsi.NewProc("MsiOpenProductW")
	procMsiQueryProductStateW   = modmsi.NewProc("MsiQueryProductStateW")
	procMsiSetInternalUI        = modmsi.NewProc("MsiSetInternalUI")
	procMsiVerifyPackageW       = modmsi.NewProc("MsiVerifyPackageW")
)

func MsiCloseHandle(handle uint32) error {
	r1, _, _ := syscall.Syscall(
		procMsiCloseHandle.Addr(),
		1,
		uintptr(handle),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiConfigureProduct(product *uint16, installLevel int32, installState int32) error {
	r1, _, _ := syscall.Syscall(
		procMsiConfigureProductW.Addr(),
		3,
		uintptr(unsafe.Pointer(product)),
		uintptr(installLevel),
		uintptr(installState))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiConfigureProductEx(product *uint16, installLevel int32, installState int32, commandLine *uint16) error {
	r1, _, _ := syscall.Syscall6(
		procMsiConfigureProductExW.Addr(),
		4,
		uintptr(unsafe.Pointer(product)),
		uintptr(installLevel),
		uintptr(installState),
		uintptr(unsafe.Pointer(commandLine)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiEnableLog(logMode uint32, logFile *uint16, logAttributes uint32) error {
	r1, _, _ := syscall.Syscall(
		procMsiEnableLogW.Addr(),
		3,
		uintptr(logMode),
		uintptr(unsafe.Pointer(logFile)),
		uintptr(logAttributes))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiEnumRelatedProducts(upgradeCode *uint16, reserved uint32, productIndex uint32, productBuf *uint16) error {
	r1, _, _ := syscall.Syscall6(
		procMsiEnumRelatedProductsW.Addr(),
		4,
		uintptr(unsafe.Pointer(upgradeCode)),
		uintptr(reserved),
		uintptr(productIndex),
		uintptr(unsafe.Pointer(productBuf)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiGetComponentPath(product *uint16, component *uint16, pathBuf *uint16, cchBuf *uint32) int32 {
	r1, _, _ := syscall.Syscall6(
		procMsiGetComponentPathW.Addr(),
		4,
		uintptr(unsafe.Pointer(product)),
		uintptr(unsafe.Pointer(component)),
		uintptr(unsafe.Pointer(pathBuf)),
		uintptr(unsafe.Pointer(cchBuf)),
		0,
		0)
	return int32(r1)
}

func MsiGetProductInfo(product *uint16, property *uint16, valueBuf *uint16, cchValueBuf *uint32) error {
	r1, _, _ := syscall.Syscall6(
		procMsiGetProductInfoW.Addr(),
		4,
		uintptr(unsafe.Pointer(product)),
		uintptr(unsafe.Pointer(property)),
		uintptr(unsafe.Pointer(valueBuf)),
		uintptr(unsafe.Pointer(cchValueBuf)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiGetProductProperty(product uint32, property *uint16, valueBuf *uint16, cchValueBuf *uint32) error {
	r1, _, _ := syscall.Syscall6(
		procMsiGetProductPropertyW.Addr(),
		4,
		uintptr(product),
		uintptr(unsafe.Pointer(property)),
		uintptr(unsafe.Pointer(valueBuf)),
		uintptr(unsafe.Pointer(cchValueBuf)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiGetProperty(install uint32, name *uint16, valueBuf *uint16, cchValueBuf *uint32) error {
	r1, _, _ := syscall.Syscall6(
		procMsiGetPropertyW.Addr(),
		4,
		uintptr(install),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(valueBuf)),
		uintptr(unsafe.Pointer(cchValueBuf)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiInstallProduct(packagePath *uint16, commandLine *uint16) error {
	r1, _, _ := syscall.Syscall(
		procMsiInstallProductW.Addr(),
		2,
		uintptr(unsafe.Pointer(packagePath)),
		uintptr(unsafe.Pointer(commandLine)),
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiOpenPackage(packagePath *uint16, product *uint32) error {
	r1, _, _ := syscall.Syscall(
		procMsiOpenPackageW.Addr(),
		2,
		uintptr(unsafe.Pointer(packagePath)),
		uintptr(unsafe.Pointer(product)),
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiOpenProduct(productCode *uint16, product *uint32) error {
	r1, _, _ := syscall.Syscall(
		procMsiOpenProductW.Addr(),
		2,
		uintptr(unsafe.Pointer(productCode)),
		uintptr(unsafe.Pointer(product)),
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func MsiQueryProductState(product *uint16) int32 {
	r1, _, _ := syscall.Syscall(
		procMsiQueryProductStateW.Addr(),
		1,
		uintptr(unsafe.Pointer(product)),
		0,
		0)
	return int32(r1)
}

func MsiSetInternalUI(uiLevel int32, window *syscall.Handle) int32 {
	r1, _, _ := syscall.Syscall(
		procMsiSetInternalUI.Addr(),
		2,
		uintptr(uiLevel),
		uintptr(unsafe.Pointer(window)),
		0)
	return int32(r1)
}

func MsiVerifyPackage(packagePath *uint16) error {
	r1, _, _ := syscall.Syscall(
		procMsiVerifyPackageW.Addr(),
		1,
		uintptr(unsafe.Pointer(packagePath)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}
