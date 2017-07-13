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
)

type SystemTimeCounters struct {
	Idle   uint64
	Kernel uint64
	User   uint64
}

func fileTimeToUint64(fileTime wrappers.FILETIME) uint64 {
	return uint64(fileTime.HighDateTime)<<32 | uint64(fileTime.LowDateTime)
}

func GetSystemTimeCounters() (*SystemTimeCounters, error) {
	var idleTime, kernelTime, userTime wrappers.FILETIME
	if err := wrappers.GetSystemTimes(&idleTime, &kernelTime, &userTime); err != nil {
		return nil, NewWindowsError("GetSystemTimes", err)
	}
	return &SystemTimeCounters{
		Idle:   fileTimeToUint64(idleTime),
		Kernel: fileTimeToUint64(kernelTime) - fileTimeToUint64(idleTime),
		User:   fileTimeToUint64(userTime),
	}, nil
}

func GetTimeCounter() uint64 {
	var systemTime wrappers.FILETIME
	wrappers.GetSystemTimeAsFileTime(&systemTime)
	return fileTimeToUint64(systemTime)
}
