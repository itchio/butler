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

func GenerateTempFileName(pathName string, prefixString string, unique uint) (string, uint, error) {
	buf := [wrappers.MAX_PATH]uint16{}
	result, err := wrappers.GetTempFileName(
		syscall.StringToUTF16Ptr(pathName),
		syscall.StringToUTF16Ptr(prefixString),
		uint32(unique),
		&buf[0])
	if err != nil {
		return "", 0, NewWindowsError("GetTempFileName", err)
	}
	return syscall.UTF16ToString(buf[:]), uint(result), nil
}

func GetTempFilePath() (string, error) {
	len, err := wrappers.GetTempPath(0, nil)
	if err != nil {
		return "", NewWindowsError("GetTempPath", err)
	}
	buf := make([]uint16, len)
	if _, err := wrappers.GetTempPath(len, &buf[0]); err != nil {
		return "", NewWindowsError("GetTempPath", err)
	}
	return syscall.UTF16ToString(buf), nil
}
