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

type RegRoot syscall.Handle

const (
	RegRootHKCR RegRoot = wrappers.HKEY_CLASSES_ROOT
	RegRootHKCU RegRoot = wrappers.HKEY_CURRENT_USER
	RegRootHKLM RegRoot = wrappers.HKEY_LOCAL_MACHINE
	RegRootHKU  RegRoot = wrappers.HKEY_USERS
	RegRootHKPD RegRoot = wrappers.HKEY_PERFORMANCE_DATA
	RegRootHKCC RegRoot = wrappers.HKEY_CURRENT_CONFIG
	RegRootHKDD RegRoot = wrappers.HKEY_DYN_DATA
)

type RegKey struct {
	handle syscall.Handle
}

func CreateRegKey(root RegRoot, subKey string) (*RegKey, error) {
	var hKey syscall.Handle
	err := wrappers.RegCreateKeyEx(
		syscall.Handle(root),
		syscall.StringToUTF16Ptr(subKey),
		0,
		nil,
		0,
		wrappers.KEY_READ | wrappers.KEY_WRITE,
		nil,
		&hKey,
		nil)
	if err != nil {
		return nil, NewWindowsError("RegCreateKeyEx", err)
	}
	return &RegKey{handle: hKey}, nil
}

func OpenRegKey(root RegRoot, subKey string, readWrite bool) (*RegKey, error) {
	var hKey syscall.Handle
	var accessMask uint32 = wrappers.KEY_READ
	if readWrite {
		accessMask |= wrappers.KEY_WRITE
	}
	err := wrappers.RegOpenKeyEx(
		syscall.Handle(root),
		syscall.StringToUTF16Ptr(subKey),
		0,
		accessMask,
		&hKey)
	if err != nil {
		return nil, NewWindowsError("RegOpenKeyEx", err)
	}
	return &RegKey{handle: hKey}, nil
}

func (self *RegKey) Close() error {
	if self.handle != 0 {
		if err := wrappers.RegCloseKey(self.handle); err != nil {
			return NewWindowsError("RegCloseKey", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *RegKey) CreateSubKey(subKey string) (*RegKey, error) {
	return CreateRegKey(RegRoot(self.handle), subKey)
}

func (self *RegKey) DeleteSubKey(subKey string) error {
	if err := wrappers.RegDeleteKey(self.handle, syscall.StringToUTF16Ptr(subKey)); err != nil {
		return NewWindowsError("RegDeleteKey", err)
	}
	return nil
}

func (self *RegKey) DeleteValue(valueName string) error {
	if err := wrappers.RegDeleteValue(self.handle, syscall.StringToUTF16Ptr(valueName)); err != nil {
		return NewWindowsError("RegDeleteValue", err)
	}
	return nil
}

func (self *RegKey) GetSubKeys() ([]string, error) {
	var subKeyCount uint32
	var maxBuffer uint32
	err := wrappers.RegQueryInfoKey(
		self.handle,
		nil,
		nil,
		nil,
		&subKeyCount,
		&maxBuffer,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil)
	if err != nil {
		return nil, NewWindowsError("RegQueryInfoKey", err)
	}
	subKeys := make([]string, 0, subKeyCount)
	buf := make([]uint16, maxBuffer)
	var i uint32
	for i = 0; i < subKeyCount; i++ {
		size := maxBuffer
		err = wrappers.RegEnumKeyEx(
			self.handle,
			i,
			&buf[0],
			&size,
			nil,
			nil,
			nil,
			nil)
		if err != nil {
			return nil, NewWindowsError("RegEnumKeyEx", err)
		}
		subKeys = append(subKeys, syscall.UTF16ToString(buf[0:size]))
	}
	return subKeys, nil
}

func (self *RegKey) GetValueBinary(valueName string) ([]byte, error) {
	var valueType uint32
	var size uint32
	err := wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		&valueType,
		nil,
		&size)
	if err != nil {
		return nil, NewWindowsError("RegQueryValueEx", err)
	}
	if valueType != wrappers.REG_BINARY {
		// use the same error code as RegGetValue, although that function is not used here in order to maintain
		// compatibility with older versions of Windows
		return nil, NewWindowsError("RegQueryValueEx", wrappers.ERROR_UNSUPPORTED_TYPE)
	}
	value := make([]byte, size)
	err = wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		nil,
		&value[0],
		&size)
	if err != nil {
		return nil, NewWindowsError("RegQueryValueEx", err)
	}
	return value, nil
}

func (self *RegKey) GetValueDWORD(valueName string) (uint32, error) {
	var valueType uint32
	var size uint32
	err := wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		&valueType,
		nil,
		&size)
	if err != nil {
		return 0, NewWindowsError("RegQueryValueEx", err)
	}
	if valueType != wrappers.REG_DWORD {
		// use the same error code as RegGetValue, although that function is not used here in order to maintain
		// compatibility with older versions of Windows
		return 0, NewWindowsError("RegQueryValueEx", wrappers.ERROR_UNSUPPORTED_TYPE)
	}
	var value uint32
	err = wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		nil,
		(*byte)(unsafe.Pointer(&value)),
		&size)
	if err != nil {
		return 0, NewWindowsError("RegQueryValueEx", err)
	}
	return value, nil
}

func (self *RegKey) GetValueQWORD(valueName string) (uint64, error) {
	var valueType uint32
	var size uint32
	err := wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		&valueType,
		nil,
		&size)
	if err != nil {
		return 0, NewWindowsError("RegQueryValueEx", err)
	}
	if valueType != wrappers.REG_QWORD {
		// use the same error code as RegGetValue, although that function is not used here in order to maintain
		// compatibility with older versions of Windows
		return 0, NewWindowsError("RegQueryValueEx", wrappers.ERROR_UNSUPPORTED_TYPE)
	}
	var value uint64
	err = wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		nil,
		(*byte)(unsafe.Pointer(&value)),
		&size)
	if err != nil {
		return 0, NewWindowsError("RegQueryValueEx", err)
	}
	return value, nil
}

func (self *RegKey) GetValueString(valueName string) (string, error) {
	var valueType uint32
	var size uint32
	err := wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		&valueType,
		nil,
		&size)
	if err != nil {
		return "", NewWindowsError("RegQueryValueEx", err)
	}
	if valueType != wrappers.REG_SZ {
		// use the same error code as RegGetValue, although that function is not used here in order to maintain
		// compatibility with older versions of Windows
		return "", NewWindowsError("RegQueryValueEx", wrappers.ERROR_UNSUPPORTED_TYPE)
	}
	buf := make([]uint16, size/2)
	err = wrappers.RegQueryValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		nil,
		nil,
		(*byte)(unsafe.Pointer(&buf[0])),
		&size)
	if err != nil {
		return "", NewWindowsError("RegQueryValueEx", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func (self *RegKey) GetValues() ([]string, error) {
	var valueCount uint32
	var maxBuffer uint32
	err := wrappers.RegQueryInfoKey(
		self.handle,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		&valueCount,
		&maxBuffer,
		nil,
		nil,
		nil)
	if err != nil {
		return nil, NewWindowsError("RegQueryInfoKey", err)
	}
	values := make([]string, 0, valueCount)
	buf := make([]uint16, maxBuffer)
	var i uint32
	for i = 0; i < valueCount; i++ {
		size := maxBuffer
		err = wrappers.RegEnumValue(
			self.handle,
			i,
			&buf[0],
			&size,
			nil,
			nil,
			nil,
			nil)
		if err != nil && err != wrappers.ERROR_MORE_DATA {
			return nil, NewWindowsError("RegEnumValue", err)
		}
		values = append(values, syscall.UTF16ToString(buf[0:size]))
	}
	return values, nil
}

func (self *RegKey) OpenSubKey(subKey string, readWrite bool) (*RegKey, error) {
	return OpenRegKey(RegRoot(self.handle), subKey, readWrite)
}

func (self *RegKey) SetValueBinary(valueName string, data []byte) error {
	err := wrappers.RegSetValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		0,
		wrappers.REG_BINARY,
		&data[0],
		uint32(len(data)))
	if err != nil {
		return NewWindowsError("RegSetValueEx", err)
	}
	return nil
}

func (self *RegKey) SetValueDWORD(valueName string, data uint32) error {
	err := wrappers.RegSetValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		0,
		wrappers.REG_DWORD,
		(*byte)(unsafe.Pointer(&data)),
		uint32(unsafe.Sizeof(data)))
	if err != nil {
		return NewWindowsError("RegSetValueEx", err)
	}
	return nil
}

func (self *RegKey) SetValueQWORD(valueName string, data uint64) error {
	err := wrappers.RegSetValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		0,
		wrappers.REG_QWORD,
		(*byte)(unsafe.Pointer(&data)),
		uint32(unsafe.Sizeof(data)))
	if err != nil {
		return NewWindowsError("RegSetValueEx", err)
	}
	return nil
}

func (self *RegKey) SetValueString(valueName string, data string) error {
	err := wrappers.RegSetValueEx(
		self.handle,
		syscall.StringToUTF16Ptr(valueName),
		0,
		wrappers.REG_SZ,
		(*byte)(unsafe.Pointer(syscall.StringToUTF16Ptr(data))),
		uint32(2*(len(data) + 1)))
	if err != nil {
		return NewWindowsError("RegSetValueEx", err)
	}
	return nil
}

func DeleteRegValue(root RegRoot, subKey string, valueName string) error {
	key, err := OpenRegKey(root, subKey, true)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.DeleteValue(valueName)
}

func GetRegValueDWORD(root RegRoot, subKey string, valueName string) (uint32, error) {
	key, err := OpenRegKey(root, subKey, false)
	if err != nil {
		return 0, err
	}
	defer key.Close()
	return key.GetValueDWORD(valueName)
}

func GetRegValueString(root RegRoot, subKey string, valueName string) (string, error) {
	key, err := OpenRegKey(root, subKey, false)
	if err != nil {
		return "", err
	}
	defer key.Close()
	return key.GetValueString(valueName)
}

func SetRegValueDWORD(root RegRoot, subKey string, valueName string, data uint32) error {
	key, err := CreateRegKey(root, subKey)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetValueDWORD(valueName, data)
}

func SetRegValueString(root RegRoot, subKey string, valueName string, data string) error {
	key, err := CreateRegKey(root, subKey)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetValueString(valueName, data)
}
