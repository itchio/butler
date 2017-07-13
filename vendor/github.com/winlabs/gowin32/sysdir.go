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

func GetSharedWindowsPath() (string, error) {
	len, err := wrappers.GetSystemWindowsDirectory(nil, 0)
	if err != nil {
		return "", NewWindowsError("GetSystemWindowsDirectory", err)
	}
	buf := make([]uint16, len)
	if _, err := wrappers.GetSystemWindowsDirectory(&buf[0], len); err != nil {
		return "", NewWindowsError("GetSystemWindowsDirectory", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowsPath() (string, error) {
	len, err := wrappers.GetWindowsDirectory(nil, 0)
	if err != nil {
		return "", NewWindowsError("GetWindowsDirectory", err)
	}
	buf := make([]uint16, len)
	if _, err := wrappers.GetWindowsDirectory(&buf[0], len); err != nil {
		return "", NewWindowsError("GetWindowsDirectory", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowsSystemPath() (string, error) {
	len, err := wrappers.GetSystemDirectory(nil, 0)
	if err != nil {
		return "", NewWindowsError("GetSystemDirectory", err)
	}
	buf := make([]uint16, len)
	if _, err := wrappers.GetSystemDirectory(&buf[0], len); err != nil {
		return "", NewWindowsError("GetSystemDirectory", err)
	}
	return syscall.UTF16ToString(buf), nil
}

func GetWindowsSystemWOW64Path() (string, error) {
	len, err := wrappers.GetSystemWow64Directory(nil, 0)
	if err != nil {
		return "", NewWindowsError("GetSystemWow64Directory", err)
	}
	buf := make([]uint16, len)
	if _, err := wrappers.GetSystemWow64Directory(&buf[0], len); err != nil {
		return "", NewWindowsError("GetSystemWow64Directory", err)
	}
	return syscall.UTF16ToString(buf), nil
}
