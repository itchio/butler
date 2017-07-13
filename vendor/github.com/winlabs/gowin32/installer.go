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

type InstallUILevel int32

const (
	InstallUILevelNoChange      InstallUILevel = wrappers.INSTALLUILEVEL_NOCHANGE
	InstallUILevelDefault       InstallUILevel = wrappers.INSTALLUILEVEL_DEFAULT
	InstallUILevelNone          InstallUILevel = wrappers.INSTALLUILEVEL_NONE
	InstallUILevelBasic         InstallUILevel = wrappers.INSTALLUILEVEL_BASIC
	InstallUILevelReduced       InstallUILevel = wrappers.INSTALLUILEVEL_REDUCED
	InstallUILevelFull          InstallUILevel = wrappers.INSTALLUILEVEL_FULL
	InstallUILevelEndDialog     InstallUILevel = wrappers.INSTALLUILEVEL_ENDDIALOG
	InstallUILevelProgressOnly  InstallUILevel = wrappers.INSTALLUILEVEL_PROGRESSONLY
	InstallUILevelHideCancel    InstallUILevel = wrappers.INSTALLUILEVEL_HIDECANCEL
	InstallUILevelSourceResOnly InstallUILevel = wrappers.INSTALLUILEVEL_SOURCERESONLY
)

type InstallState int32

const (
	InstallStateBadConfig    InstallState = wrappers.INSTALLSTATE_BADCONFIG
	InstallStateIncomplete   InstallState = wrappers.INSTALLSTATE_INCOMPLETE
	InstallStateSourceAbsent InstallState = wrappers.INSTALLSTATE_SOURCEABSENT
	InstallStateMoreData     InstallState = wrappers.INSTALLSTATE_MOREDATA
	InstallStateInvalidArg   InstallState = wrappers.INSTALLSTATE_INVALIDARG
	InstallStateUnknown      InstallState = wrappers.INSTALLSTATE_UNKNOWN
	InstallStateBroken       InstallState = wrappers.INSTALLSTATE_BROKEN
	InstallStateAdvertised   InstallState = wrappers.INSTALLSTATE_ADVERTISED
	InstallStateAbsent       InstallState = wrappers.INSTALLSTATE_ABSENT
	InstallStateLocal        InstallState = wrappers.INSTALLSTATE_LOCAL
	InstallStateSource       InstallState = wrappers.INSTALLSTATE_SOURCE
	InstallStateDefault      InstallState = wrappers.INSTALLSTATE_DEFAULT
)

type InstallLevel int32

const (
	InstallLevelDefault InstallLevel = wrappers.INSTALLLEVEL_DEFAULT
	InstallLevelMinimum InstallLevel = wrappers.INSTALLLEVEL_MINIMUM
	InstallLevelMaximum InstallLevel = wrappers.INSTALLLEVEL_MAXIMUM
)

type InstallLogMode uint32

const (
	InstallLogModeFatalExit      InstallLogMode = wrappers.INSTALLLOGMODE_FATALEXIT
	InstallLogModeError          InstallLogMode = wrappers.INSTALLLOGMODE_ERROR
	InstallLogModeWarning        InstallLogMode = wrappers.INSTALLLOGMODE_WARNING
	InstallLogModeUser           InstallLogMode = wrappers.INSTALLLOGMODE_USER
	InstallLogModeInfo           InstallLogMode = wrappers.INSTALLLOGMODE_INFO
	InstallLogModeResolveSource  InstallLogMode = wrappers.INSTALLLOGMODE_RESOLVESOURCE
	InstallLogModeOutOfDiskSpace InstallLogMode = wrappers.INSTALLLOGMODE_OUTOFDISKSPACE
	InstallLogModeActionStart    InstallLogMode = wrappers.INSTALLLOGMODE_ACTIONSTART
	InstallLogModeActionData     InstallLogMode = wrappers.INSTALLLOGMODE_ACTIONDATA
	InstallLogModeCommonData     InstallLogMode = wrappers.INSTALLLOGMODE_COMMONDATA
	InstallLogModePropertyDump   InstallLogMode = wrappers.INSTALLLOGMODE_PROPERTYDUMP
	InstallLogModeVerbose        InstallLogMode = wrappers.INSTALLLOGMODE_VERBOSE
	InstallLogModeExtraDebug     InstallLogMode = wrappers.INSTALLLOGMODE_EXTRADEBUG
	InstallLogModeLogOnlyOnError InstallLogMode = wrappers.INSTALLLOGMODE_LOGONLYONERROR
	InstallLogModeProgress       InstallLogMode = wrappers.INSTALLLOGMODE_PROGRESS
	InstallLogModeInitialize     InstallLogMode = wrappers.INSTALLLOGMODE_INITIALIZE
	InstallLogModeTerminate      InstallLogMode = wrappers.INSTALLLOGMODE_TERMINATE
	InstallLogModeShowDialog     InstallLogMode = wrappers.INSTALLLOGMODE_SHOWDIALOG
	InstallLogModeFilesInUse     InstallLogMode = wrappers.INSTALLLOGMODE_FILESINUSE
	InstallLogModeRMFilesInUse   InstallLogMode = wrappers.INSTALLLOGMODE_RMFILESINUSE
	InstallLogModeInstallStart   InstallLogMode = wrappers.INSTALLLOGMODE_INSTALLSTART
	InstallLogModeInstallEnd     InstallLogMode = wrappers.INSTALLLOGMODE_INSTALLEND
)

type InstallLogAttributes uint32

const (
	InstallLogAttributesAppend        InstallLogAttributes = wrappers.INSTALLLOGATTRIBUTES_APPEND
	InstallLogAttributesFlushEachLine InstallLogAttributes = wrappers.INSTALLLOGATTRIBUTES_FLUSHEACHLINE
)

type InstallProperty string

const (
	InstallPropertyPackageName          InstallProperty = wrappers.INSTALLPROPERTY_PACKAGENAME
	InstallPropertyTransforms           InstallProperty = wrappers.INSTALLPROPERTY_TRANSFORMS
	InstallPropertyLanguage             InstallProperty = wrappers.INSTALLPROPERTY_LANGUAGE
	InstallPropertyProductName          InstallProperty = wrappers.INSTALLPROPERTY_PRODUCTNAME
	InstallPropertyAssignmentType       InstallProperty = wrappers.INSTALLPROPERTY_ASSIGNMENTTYPE
	InstallPropertyInstanceType         InstallProperty = wrappers.INSTALLPROPERTY_INSTANCETYPE
	InstallPropertyAuthorizedLUAApp     InstallProperty = wrappers.INSTALLPROPERTY_AUTHORIZED_LUA_APP
	InstallPropertyPackageCode          InstallProperty = wrappers.INSTALLPROPERTY_PACKAGECODE
	InstallPropertyVersion              InstallProperty = wrappers.INSTALLPROPERTY_VERSION
	InstallPropertyProductIcon          InstallProperty = wrappers.INSTALLPROPERTY_PRODUCTICON
	InstallPropertyInstalledProductName InstallProperty = wrappers.INSTALLPROPERTY_INSTALLEDPRODUCTNAME
	InstallPropertyVersionString        InstallProperty = wrappers.INSTALLPROPERTY_VERSIONSTRING
	InstallPropertyHelpLink             InstallProperty = wrappers.INSTALLPROPERTY_HELPLINK
	InstallPropertyHelpTelephone        InstallProperty = wrappers.INSTALLPROPERTY_HELPTELEPHONE
	InstallPropertyInstallLocation      InstallProperty = wrappers.INSTALLPROPERTY_INSTALLLOCATION
	InstallPropertyInstallSource        InstallProperty = wrappers.INSTALLPROPERTY_INSTALLSOURCE
	InstallPropertyInstallDate          InstallProperty = wrappers.INSTALLPROPERTY_INSTALLDATE
	InstallPropertyPublisher            InstallProperty = wrappers.INSTALLPROPERTY_PUBLISHER
	InstallPropertyLocalPackage         InstallProperty = wrappers.INSTALLPROPERTY_LOCALPACKAGE
	InstallPropertyURLInfoAbout         InstallProperty = wrappers.INSTALLPROPERTY_URLINFOABOUT
	InstallPropertyURLUpdateInfo        InstallProperty = wrappers.INSTALLPROPERTY_URLUPDATEINFO
	InstallPropertyVersionMinor         InstallProperty = wrappers.INSTALLPROPERTY_VERSIONMINOR
	InstallPropertyVersionMajor         InstallProperty = wrappers.INSTALLPROPERTY_VERSIONMAJOR
	InstallPropertyProductID            InstallProperty = wrappers.INSTALLPROPERTY_PRODUCTID
	InstallPropertyRegCompany           InstallProperty = wrappers.INSTALLPROPERTY_REGCOMPANY
	InstallPropertyRegOwner             InstallProperty = wrappers.INSTALLPROPERTY_REGOWNER
	InstallPropertyInstalledLanguage    InstallProperty = wrappers.INSTALLPROPERTY_INSTALLEDLANGUAGE
)

type InstallerPackage struct {
	handle uint32
}

func OpenInstalledProduct(productCode string) (*InstallerPackage, error) {
	var handle uint32
	if err := wrappers.MsiOpenProduct(syscall.StringToUTF16Ptr(productCode), &handle); err != nil {
		return nil, NewWindowsError("MsiOpenProduct", err)
	}
	return &InstallerPackage{handle: handle}, nil
}

func OpenInstallerPackage(packagePath string) (*InstallerPackage, error) {
	var handle uint32
	if err := wrappers.MsiOpenPackage(syscall.StringToUTF16Ptr(packagePath), &handle); err != nil {
		return nil, NewWindowsError("MsiOpenPackage", err)
	}
	return &InstallerPackage{handle: handle}, nil
}

func (self *InstallerPackage) Close() error {
	if self.handle != 0 {
		if err := wrappers.MsiCloseHandle(self.handle); err != nil {
			return NewWindowsError("MsiCloseHandle", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *InstallerPackage) GetProductProperty(property string) (string, error) {
	var size uint32
	err := wrappers.MsiGetProductProperty(
		self.handle,
		syscall.StringToUTF16Ptr(property),
		nil,
		&size)
	if err != nil {
		return "", NewWindowsError("MsiGetProductProperty", err)
	}
	size++
	buf := make([]uint16, size)
	err = wrappers.MsiGetProductProperty(
		self.handle,
		syscall.StringToUTF16Ptr(property),
		&buf[0],
		&size)
	if err != nil {
		return "", NewWindowsError("MsiGetProductProperty", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func (self *InstallerPackage) GetProperty(name string) (string, error) {
	var size uint32
	err := wrappers.MsiGetProperty(
		self.handle,
		syscall.StringToUTF16Ptr(name),
		syscall.StringToUTF16Ptr(""),
		&size)
	if err != wrappers.ERROR_MORE_DATA {
		return "", NewWindowsError("MsiGetProperty", err)
	}
	size++
	buf := make([]uint16, size)
	err = wrappers.MsiGetProperty(
		self.handle,
		syscall.StringToUTF16Ptr(name),
		&buf[0],
		&size)
	if err != nil {
		return "", NewWindowsError("MsiGetProperty", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func ConfigureInstalledProduct(productCode string, installLevel InstallLevel, installState InstallState, commandLine string) error {
	err := wrappers.MsiConfigureProductEx(
		syscall.StringToUTF16Ptr(productCode),
		int32(installLevel),
		int32(installState),
		syscall.StringToUTF16Ptr(commandLine))
	if err != nil {
		return NewWindowsError("MsiConfigureProductEx", err)
	}
	return nil
}

func DisableInstallerLog() error {
	err := wrappers.MsiEnableLog(0, nil, 0)
	if err != nil {
		return NewWindowsError("MsiEnableLog", err)
	}
	return nil
}

func EnableInstallerLog(logMode InstallLogMode, logFile string, logAttributes InstallLogAttributes) error {
	err := wrappers.MsiEnableLog(
		uint32(logMode),
		syscall.StringToUTF16Ptr(logFile),
		uint32(logAttributes))
	if err != nil {
		return NewWindowsError("MsiEnableLog", err)
	}
	return nil
}

func GetInstalledComponentPath(productCode string, componentID string) (string, InstallState) {
	var size uint32
	wrappers.MsiGetComponentPath(
		syscall.StringToUTF16Ptr(productCode),
		syscall.StringToUTF16Ptr(componentID),
		nil,
		&size)
	size++
	buf := make([]uint16, size)
	state := wrappers.MsiGetComponentPath(
		syscall.StringToUTF16Ptr(productCode),
		syscall.StringToUTF16Ptr(componentID),
		&buf[0],
		&size)
	return syscall.UTF16ToString(buf), InstallState(state)
}

func GetInstalledProductProperty(productCode string, property InstallProperty) (string, error) {
	var size uint32
	err := wrappers.MsiGetProductInfo(
		syscall.StringToUTF16Ptr(productCode),
		syscall.StringToUTF16Ptr(string(property)),
		nil,
		&size)
	if err != nil {
		return "", NewWindowsError("MsiGetProductInfo", err)
	}
	size++
	buf := make([]uint16, size)
	err = wrappers.MsiGetProductInfo(
		syscall.StringToUTF16Ptr(productCode),
		syscall.StringToUTF16Ptr(string(property)),
		&buf[0],
		&size)
	if err != nil {
		return "", NewWindowsError("MsiGetProductInfo", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func GetInstalledProductState(productCode string) InstallState {
	return InstallState(wrappers.MsiQueryProductState(syscall.StringToUTF16Ptr(productCode)))
}

func GetInstalledProductsByUpgradeCode(upgradeCode string) ([]string, error) {
	productCodes := []string{}
	buf := make([]uint16, 39)
	for i := 0; ; i++ {
		err := wrappers.MsiEnumRelatedProducts(syscall.StringToUTF16Ptr(upgradeCode), 0, uint32(i), &buf[0])
		if err == wrappers.ERROR_NO_MORE_ITEMS {
			return productCodes, nil
		} else if err != nil {
			return nil, NewWindowsError("MsiEnumRelatedProducts", err)
		}
		productCodes = append(productCodes, syscall.UTF16ToString(buf))
	}
}

func InstallProduct(packagePath string, commandLine string) error {
	err := wrappers.MsiInstallProduct(
		syscall.StringToUTF16Ptr(packagePath),
		syscall.StringToUTF16Ptr(commandLine))
	if err != nil {
		return NewWindowsError("MsiInstallProduct", err)
	}
	return nil
}

func SetInstallerInternalUI(uiLevel InstallUILevel) InstallUILevel {
	return InstallUILevel(wrappers.MsiSetInternalUI(int32(uiLevel), nil))
}

func UninstallProduct(productCode string) error {
	err := wrappers.MsiConfigureProduct(
		syscall.StringToUTF16Ptr(productCode),
		0,
		wrappers.INSTALLSTATE_ABSENT)
	if err != nil {
		return NewWindowsError("MsiConfigureProduct", err)
	}
	return nil
}

func VerifyInstallerPackage(packagePath string) (bool, error) {
	if err := wrappers.MsiVerifyPackage(syscall.StringToUTF16Ptr(packagePath)); err != nil {
		if err == wrappers.ERROR_INSTALL_PACKAGE_INVALID {
			return false, nil
		} else {
			return false, NewWindowsError("MsiVerifyPackage", err)
		}
	}
	return true, nil
}
