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

	"path/filepath"
	"syscall"
	"unsafe"
)

type ReparseTag uint32

const (
	ReparseTagMountPoint   ReparseTag = wrappers.IO_REPARSE_TAG_MOUNT_POINT
	ReparseTagHSM          ReparseTag = wrappers.IO_REPARSE_TAG_HSM
	ReparseTagHSM2         ReparseTag = wrappers.IO_REPARSE_TAG_HSM2
	ReparseTagSIS          ReparseTag = wrappers.IO_REPARSE_TAG_SIS
	ReparseTagCSV          ReparseTag = wrappers.IO_REPARSE_TAG_CSV
	ReparseTagDFS          ReparseTag = wrappers.IO_REPARSE_TAG_DFS
	ReparseTagSymbolicLink ReparseTag = wrappers.IO_REPARSE_TAG_SYMLINK
	ReparseTagDFSR         ReparseTag = wrappers.IO_REPARSE_TAG_DFSR
)

type FindFileItem struct {
	FileAttributes    FileAttributes
	FileSize          uint64
	ReparseTag        ReparseTag
	FileName          string
	AlternateFileName string
}

type FindFile struct {
	handle   syscall.Handle
	fileName string
	current  wrappers.WIN32_FIND_DATA
}

func OpenFindFile(fileName string) *FindFile {
	return &FindFile{fileName: fileName}
}

func (self *FindFile) Close() error {
	if self.handle != 0 {
		if err := wrappers.FindClose(self.handle); err != nil {
			return NewWindowsError("FindClose", err)
		}
		self.handle = 0
		wrappers.RtlZeroMemory((*byte)(unsafe.Pointer(&self.current)), unsafe.Sizeof(self.current))
	}
	return nil
}

func (self *FindFile) Current() FindFileItem {
	return FindFileItem{
		FileAttributes:    FileAttributes(self.current.FileAttributes),
		FileSize:          (uint64(self.current.FileSizeHigh) << 32) | uint64(self.current.FileSizeLow),
		ReparseTag:        ReparseTag(self.current.Reserved0),
		FileName:          syscall.UTF16ToString(self.current.FileName[:]),
		AlternateFileName: syscall.UTF16ToString(self.current.AlternateFileName[:]),
	}
}

func (self *FindFile) Next() (bool, error) {
	if self.handle == 0 {
		handle, err := wrappers.FindFirstFile(syscall.StringToUTF16Ptr(self.fileName), &self.current)
		if err == wrappers.ERROR_FILE_NOT_FOUND {
			return false, nil
		} else if err != nil {
			return false, NewWindowsError("FindFirstFile", err)
		}
		self.handle = handle
	} else {
		if err := wrappers.FindNextFile(self.handle, &self.current); err == wrappers.ERROR_NO_MORE_FILES {
			return false, nil
		} else if err != nil {
			return false, NewWindowsError("FindNextFile", err)
		}
	}
	return true, nil
}

func GetDirectorySize(dirName string) (uint64, error) {
	wildcard := filepath.Join(dirName, "*.*")
	ff := OpenFindFile(wildcard)
	defer ff.Close()
	var totalSize uint64
	for {
		if more, err := ff.Next(); err != nil {
			return 0, err
		} else if !more {
			break
		}
		info := ff.Current()
		if info.FileName == "." || info.FileName == ".." {
			continue
		} else if (info.FileAttributes & FileAttributeDirectory) != 0 {
			subdir := filepath.Join(dirName, info.FileName)
			subdirSize, err := GetDirectorySize(subdir)
			if err != nil {
				return 0, err
			}
			totalSize += subdirSize
		} else {
			totalSize += info.FileSize
		}
	}
	return totalSize, nil
}

func getDirectorySizeOnDisk(dirName string, clusterSize uint64) (uint64, error) {
	wildcard := filepath.Join(dirName, "*.*")
	ff := OpenFindFile(wildcard)
	defer ff.Close()
	var totalSize uint64
	for {
		if more, err := ff.Next(); err != nil {
			return 0, err
		} else if !more {
			break
		}
		info := ff.Current()
		if info.FileName == "." || info.FileName == ".." {
			continue
		} else if (info.FileAttributes & FileAttributeDirectory) != 0 {
			subdir := filepath.Join(dirName, info.FileName)
			subdirSize, err := getDirectorySizeOnDisk(subdir, clusterSize)
			if err != nil {
				return 0, err
			}
			totalSize += subdirSize
		} else if (info.FileAttributes & FileAttributeCompressed) != 0 {
			fileName := filepath.Join(dirName, info.FileName)
			compressedSize, err := GetCompressedSize(fileName)
			if err != nil {
				return 0, err
			}
			totalSize += compressedSize
		} else if (info.FileSize % clusterSize) != 0 {
			totalSize += info.FileSize - (info.FileSize % clusterSize) + clusterSize
		} else {
			totalSize += info.FileSize
		}
	}
	return totalSize, nil
}

func GetDirectorySizeOnDisk(dirName string) (uint64, error) {
	volume, err := GetVolumePath(dirName)
	if err != nil {
		return 0, err
	}
	sectorsPerCluster, bytesPerSector, _, _, err := GetSectorsAndClusters(volume)
	if err != nil {
		return 0, err
	}
	return getDirectorySizeOnDisk(dirName, uint64(sectorsPerCluster) * uint64(bytesPerSector))
}
