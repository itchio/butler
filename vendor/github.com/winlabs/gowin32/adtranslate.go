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

type ADNameType int32

const (
	ADNameType1779                 ADNameType = wrappers.ADS_NAME_TYPE_1779
	ADNameTypeCanonical            ADNameType = wrappers.ADS_NAME_TYPE_CANONICAL
	ADNameTypeNT4                  ADNameType = wrappers.ADS_NAME_TYPE_NT4
	ADNameTypeDisplay              ADNameType = wrappers.ADS_NAME_TYPE_DISPLAY
	ADNameTypeDomainSimple         ADNameType = wrappers.ADS_NAME_TYPE_DOMAIN_SIMPLE
	ADNameTypeEnterpriseSimple     ADNameType = wrappers.ADS_NAME_TYPE_ENTERPRISE_SIMPLE
	ADNameTypeGUID                 ADNameType = wrappers.ADS_NAME_TYPE_GUID
	ADNameTypeUnknown              ADNameType = wrappers.ADS_NAME_TYPE_UNKNOWN
	ADNameTypeUserPrincipalName    ADNameType = wrappers.ADS_NAME_TYPE_USER_PRINCIPAL_NAME
	ADNameTypeCanonicalEx          ADNameType = wrappers.ADS_NAME_TYPE_CANONICAL_EX
	ADNameTypeServicePrincipalName ADNameType = wrappers.ADS_NAME_TYPE_SERVICE_PRINCIPAL_NAME
	ADNameTypeSIDOrSIDHistoryName  ADNameType = wrappers.ADS_NAME_TYPE_SID_OR_SID_HISTORY_NAME
)

func TranslateADName(name string, fromType ADNameType, toType ADNameType) (string, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_NameTranslate,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_IADsNameTranslate,
		&object)
	if wrappers.FAILED(hr) {
		return "", NewWindowsError("CoCreateInstance", COMError(hr))
	}
	trans := (*wrappers.IADsNameTranslate)(unsafe.Pointer(object))
	defer trans.Release()
	if hr := trans.Init(wrappers.ADS_NAME_INITTYPE_GC, nil); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsNameTranslate::Init", COMError(hr))
	}
	nameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(name))
	defer wrappers.SysFreeString(nameRaw)
	if hr := trans.Set(int32(fromType), nameRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsNameTranslate::Set", COMError(hr))
	}
	var outRaw *uint16
	if hr := trans.Get(int32(toType), &outRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("IADsNameTranslate::Get", COMError(hr))
	}
	return BstrToString(outRaw), nil
}
