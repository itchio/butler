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

	"strings"
	"syscall"
	"unsafe"
)

func ExpandEnvironment(text string) (string, error) {
	size, err := wrappers.ExpandEnvironmentStrings(syscall.StringToUTF16Ptr(text), nil, 0)
	if err != nil {
		return "", NewWindowsError("ExpandEnvironmentStrings", err)
	}
	buf := make([]uint16, size)
	if _, err := wrappers.ExpandEnvironmentStrings(syscall.StringToUTF16Ptr(text), &buf[0], size); err != nil {
		return "", NewWindowsError("ExpandEnvironmentStrings", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func GetAllEnvironment() (map[string]string, error) {
	block, err := wrappers.GetEnvironmentStrings()
	if err != nil {
		return nil, NewWindowsError("GetEnvironmentStrings", err)
	}
	defer wrappers.FreeEnvironmentStrings(block)
	blockMap := make(map[string]string)
	item := block
	for {
		entry := LpstrToString(item)
		if len(entry) == 0 {
			return blockMap, nil
		}
		if entry[0] != '=' {
			index := strings.Index(entry, "=")
			name := entry[0:index]
			value := entry[index+1:]
			blockMap[name] = value
		}
		offset := uintptr(2*len(entry) + 2)
		item = (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(item)) + offset))
	}
}

func GetEnvironment(name string) (string, error) {
	len, err := wrappers.GetEnvironmentVariable(syscall.StringToUTF16Ptr(name), nil, 0)
	if err != nil {
		return "", NewWindowsError("GetEnvironmentVariable", err)
	}
	buf := make([]uint16, len)
	_, err = wrappers.GetEnvironmentVariable(syscall.StringToUTF16Ptr(name), &buf[0], len)
	if err != nil {
		return "", NewWindowsError("GetEnvironmentVariable", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func SetEnvironment(name string, value string) error {
	err := wrappers.SetEnvironmentVariable(
		syscall.StringToUTF16Ptr(name),
		syscall.StringToUTF16Ptr(value))
	if err != nil {
		return NewWindowsError("SetEnvironmentVariable", err)
	}
	return nil
}
