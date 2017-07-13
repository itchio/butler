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

type DiskPerformanceInfo struct {
	BytesRead           int64
	BytesWritten        int64
	ReadTime            int64
	WriteTime           int64
	IdleTime            int64
	ReadCount           uint
	WriteCount          uint
	QueueDepth          uint
	SplitCount          uint
	QueryTime           int64
	StorageDeviceNumber uint
	StorageManagerName  string
}

func GetDiskPerformanceInfo(rootPathName string) (*DiskPerformanceInfo, error) {
	hFile, err := wrappers.CreateFile(
		syscall.StringToUTF16Ptr(rootPathName),
		0,
		wrappers.FILE_SHARE_READ | wrappers.FILE_SHARE_WRITE,
		nil,
		wrappers.OPEN_EXISTING,
		0,
		0)
	if err != nil {
		return nil, NewWindowsError("CreateFile", err)
	}
	defer wrappers.CloseHandle(hFile)
	var diskPerformance wrappers.DISK_PERFORMANCE
	var diskPerformanceSize uint32
	err = wrappers.DeviceIoControl(
		hFile,
		wrappers.IOCTL_DISK_PERFORMANCE,
		nil,
		0,
		(*byte)(unsafe.Pointer(&diskPerformance)),
		uint32(unsafe.Sizeof(diskPerformance)),
		&diskPerformanceSize,
		nil)
	if err != nil {
		return nil, NewWindowsError("DeviceIoControl", err)
	}
	return &DiskPerformanceInfo{
		BytesRead:           diskPerformance.BytesRead,
		BytesWritten:        diskPerformance.BytesWritten,
		ReadTime:            diskPerformance.ReadTime,
		WriteTime:           diskPerformance.WriteTime,
		IdleTime:            diskPerformance.IdleTime,
		ReadCount:           uint(diskPerformance.ReadCount),
		WriteCount:          uint(diskPerformance.WriteCount),
		QueueDepth:          uint(diskPerformance.QueueDepth),
		SplitCount:          uint(diskPerformance.SplitCount),
		QueryTime:           diskPerformance.QueryTime,
		StorageDeviceNumber: uint(diskPerformance.StorageDeviceNumber),
		StorageManagerName:  syscall.UTF16ToString(diskPerformance.StorageManagerName[:]),
	}, nil
}
