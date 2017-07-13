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

func GetCurrentExePath() (string, error) {
	buf := make([]uint16, wrappers.MAX_PATH)
	if _, err := wrappers.GetModuleFileName(0, &buf[0], wrappers.MAX_PATH); err != nil {
		if err == wrappers.ERROR_INSUFFICIENT_BUFFER {
			buf = make([]uint16, syscall.MAX_LONG_PATH)
			if _, err := wrappers.GetModuleFileName(0, &buf[0], syscall.MAX_LONG_PATH); err != nil {
				return "", NewWindowsError("GetModuleFileName", err)
			}
		}  else {
			return "", NewWindowsError("GetModuleFileName", err)
		}
	}
	return syscall.UTF16ToString(buf), nil
}
