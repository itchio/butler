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

func GetAvailableDiskSpace(root string) (uint64, error) {
	availableSpace := uint64(0)
	if err := wrappers.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(root), &availableSpace, nil, nil); err != nil {
		return 0, NewWindowsError("GetDiskFreeSpaceEx", err)
	}
	return availableSpace, nil
}

func GetTotalDiskSpace(root string) (uint64, error) {
	totalSpace := uint64(0)
	if err := wrappers.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(root), nil, &totalSpace, nil); err != nil {
		return 0, NewWindowsError("GetDiskFreeSpaceEx", err)
	}
	return totalSpace, nil
}

func GetFreeDiskSpace(root string) (uint64, error) {
	freeSpace := uint64(0)
	if err := wrappers.GetDiskFreeSpaceEx(syscall.StringToUTF16Ptr(root), nil, nil, &freeSpace); err != nil {
		return 0, NewWindowsError("GetDiskFreeSpaceEx", err)
	}
	return freeSpace, nil
}

func GetSectorsAndClusters(root string) (uint32, uint32, uint32, uint32, error) {
	var sectorsPerCluster uint32
	var bytesPerSector uint32
	var numberOfFreeClusters uint32
	var totalNumberOfClusters uint32
	err := wrappers.GetDiskFreeSpace(
		syscall.StringToUTF16Ptr(root),
		&sectorsPerCluster,
		&bytesPerSector,
		&numberOfFreeClusters,
		&totalNumberOfClusters)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return sectorsPerCluster, bytesPerSector, numberOfFreeClusters, totalNumberOfClusters, nil
}
