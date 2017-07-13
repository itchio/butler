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

type FileSystemFlags uint32

const (
	FileSystemCaseSensitiveSearch        FileSystemFlags = wrappers.FILE_CASE_SENSITIVE_SEARCH
	FileSystemCasePreservedNames         FileSystemFlags = wrappers.FILE_CASE_PRESERVED_NAMES
	FileSystemUnicodeOnDisk              FileSystemFlags = wrappers.FILE_UNICODE_ON_DISK
	FileSystemPersistentACLs             FileSystemFlags = wrappers.FILE_PERSISTENT_ACLS
	FileSystemFileCompression            FileSystemFlags = wrappers.FILE_FILE_COMPRESSION
	FileSystemVolumeQuotas               FileSystemFlags = wrappers.FILE_VOLUME_QUOTAS
	FileSystemSupportsSparseFiles        FileSystemFlags = wrappers.FILE_SUPPORTS_SPARSE_FILES
	FileSystemSupportsReparsePoints      FileSystemFlags = wrappers.FILE_SUPPORTS_REPARSE_POINTS
	FileSystemSupportsRemoteStorage      FileSystemFlags = wrappers.FILE_SUPPORTS_REMOTE_STORAGE
	FileSystemVolumeIsCompressed         FileSystemFlags = wrappers.FILE_VOLUME_IS_COMPRESSED
	FileSystemSupportsObjectIDs          FileSystemFlags = wrappers.FILE_SUPPORTS_OBJECT_IDS
	FileSystemSupportsEncryption         FileSystemFlags = wrappers.FILE_SUPPORTS_ENCRYPTION
	FileSystemNamedStreams               FileSystemFlags = wrappers.FILE_NAMED_STREAMS
	FileSystemReadOnlyVolume             FileSystemFlags = wrappers.FILE_READ_ONLY_VOLUME
	FileSystemSequentialWriteOnce        FileSystemFlags = wrappers.FILE_SEQUENTIAL_WRITE_ONCE
	FileSystemSupportsTransactions       FileSystemFlags = wrappers.FILE_SUPPORTS_TRANSACTIONS
	FileSystemSupportsHardLinks          FileSystemFlags = wrappers.FILE_SUPPORTS_HARD_LINKS
	FileSystemSupportsExtendedAttributes FileSystemFlags = wrappers.FILE_SUPPORTS_EXTENDED_ATTRIBUTES
	FileSystemSupportsOpenByFileID       FileSystemFlags = wrappers.FILE_SUPPORTS_OPEN_BY_FILE_ID
	FileSystemSupportsUSNJournal         FileSystemFlags = wrappers.FILE_SUPPORTS_USN_JOURNAL
)

type VolumeInfo struct {
	VolumeName             string
	VolumeSerialNumber     uint
	MaximumComponentLength uint
	FileSystemFlags        FileSystemFlags
	FileSystemName         string
}

type DriveType uint32

const (
	DriveTypeUnknown DriveType = wrappers.DRIVE_UNKNOWN
	DriveNoRootDir   DriveType = wrappers.DRIVE_NO_ROOT_DIR
	DriveRemovable   DriveType = wrappers.DRIVE_REMOVABLE
	DriveFixed       DriveType = wrappers.DRIVE_FIXED
	DriveRemote      DriveType = wrappers.DRIVE_REMOTE
	DriveCDROM       DriveType = wrappers.DRIVE_CDROM
	DriveRAMDisk     DriveType = wrappers.DRIVE_RAMDISK
)

func GetVolumeInfo(rootPathName string) (*VolumeInfo, error) {
	var volumeSerialNumber uint32
	var maximumComponentLength uint32
	var fileSystemFlags uint32
	volumeNameBuffer := make([]uint16, syscall.MAX_PATH+1)
	fileSystemNameBuffer := make([]uint16, syscall.MAX_PATH+1)
	err := wrappers.GetVolumeInformation(
		syscall.StringToUTF16Ptr(rootPathName),
		&volumeNameBuffer[0],
		syscall.MAX_PATH+1,
		&volumeSerialNumber,
		&maximumComponentLength,
		&fileSystemFlags,
		&fileSystemNameBuffer[0],
		syscall.MAX_PATH+1)
	if err != nil {
		return nil, NewWindowsError("GetVolumeInformation", err)
	}
	return &VolumeInfo{
		VolumeName:             syscall.UTF16ToString(volumeNameBuffer),
		VolumeSerialNumber:     uint(volumeSerialNumber),
		MaximumComponentLength: uint(maximumComponentLength),
		FileSystemFlags:        FileSystemFlags(fileSystemFlags),
		FileSystemName:         syscall.UTF16ToString(fileSystemNameBuffer),
	}, nil
}

func GetVolumeDriveType(rootPathName string) DriveType {
	return DriveType(wrappers.GetDriveType(syscall.StringToUTF16Ptr(rootPathName)))
}

const (
	MaximumVolumeGUIDPath = 50
)

func GetVolumeNameFromMountPoint(volumeMountPoint string) (string, error) {
	volumeName := make([]uint16, MaximumVolumeGUIDPath)

	err := wrappers.GetVolumeNameForVolumeMountPoint(
		syscall.StringToUTF16Ptr(volumeMountPoint),
		&volumeName[0],
		MaximumVolumeGUIDPath)

	if err != nil {
		return "", NewWindowsError("GetVolumeNameForVolumeMountPoint", err)
	}
	return syscall.UTF16ToString(volumeName), nil
}
