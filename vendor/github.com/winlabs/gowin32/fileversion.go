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

	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

type VerFileFlags uint32

const (
	VerFileDebug        VerFileFlags = wrappers.VS_FF_DEBUG
	VerFilePrerelease   VerFileFlags = wrappers.VS_FF_PRERELEASE
	VerFilePatched      VerFileFlags = wrappers.VS_FF_PATCHED
	VerFilePrivateBuild VerFileFlags = wrappers.VS_FF_PRIVATEBUILD
	VerFileInfoInferred VerFileFlags = wrappers.VS_FF_INFOINFERRED
	VerFileSpecialBuild VerFileFlags = wrappers.VS_FF_SPECIALBUILD
)

type VerFileOS uint32

const (
	VerFileOSUnknown    VerFileOS = wrappers.VOS_UNKNOWN
	VerFileDOS          VerFileOS = wrappers.VOS_DOS
	VerFileOS216        VerFileOS = wrappers.VOS_OS216
	VerFileOS232        VerFileOS = wrappers.VOS_OS232
	VerFileNT           VerFileOS = wrappers.VOS_NT
	VerFileWindows16    VerFileOS = wrappers.VOS__WINDOWS16
	VerFilePM16         VerFileOS = wrappers.VOS__PM16
	VerFilePM32         VerFileOS = wrappers.VOS__PM32
	VerFileWindows32    VerFileOS = wrappers.VOS__WINDOWS32
	VerFileDOSWindows16 VerFileOS = wrappers.VOS_DOS_WINDOWS16
	VerFileDOSWindows32 VerFileOS = wrappers.VOS_DOS_WINDOWS32
	VerFileOS216PM16    VerFileOS = wrappers.VOS_OS216_PM16
	VerFileOS232PM32    VerFileOS = wrappers.VOS_OS232_PM32
	VerFileNTWindows32  VerFileOS = wrappers.VOS_NT_WINDOWS32
)

type VerFileType uint32

const (
	VerFileTypeUnknown VerFileType = wrappers.VFT_UNKNOWN
	VerFileApp         VerFileType = wrappers.VFT_APP
	VerFileDLL         VerFileType = wrappers.VFT_DLL
	VerFileDriver      VerFileType = wrappers.VFT_DRV
	VerFileFont        VerFileType = wrappers.VFT_FONT
	VerFileVxD         VerFileType = wrappers.VFT_VXD
	VerFileStaticLib   VerFileType = wrappers.VFT_STATIC_LIB
)

type VerFileSubtype uint32

const (
	VerFileSubtypeUnknown         VerFileSubtype = wrappers.VFT2_UNKNOWN
	VerFileDriverPrinter          VerFileSubtype = wrappers.VFT2_DRV_PRINTER
	VerFileDriverKeyboard         VerFileSubtype = wrappers.VFT2_DRV_KEYBOARD
	VerFileDriverLanguage         VerFileSubtype = wrappers.VFT2_DRV_LANGUAGE
	VerFileDriverDisplay          VerFileSubtype = wrappers.VFT2_DRV_DISPLAY
	VerFileDriverMouse            VerFileSubtype = wrappers.VFT2_DRV_MOUSE
	VerFileDriverNetwork          VerFileSubtype = wrappers.VFT2_DRV_NETWORK
	VerFileDriverSystem           VerFileSubtype = wrappers.VFT2_DRV_SYSTEM
	VerFileDriverInstallable      VerFileSubtype = wrappers.VFT2_DRV_INSTALLABLE
	VerFileDriverSound            VerFileSubtype = wrappers.VFT2_DRV_SOUND
	VerFileDriverComm             VerFileSubtype = wrappers.VFT2_DRV_COMM
	VerFileDriverVersionedPrinter VerFileSubtype = wrappers.VFT2_DRV_VERSIONED_PRINTER
	VerFileFontRaster             VerFileSubtype = wrappers.VFT2_FONT_RASTER
	VerFileFontVector             VerFileSubtype = wrappers.VFT2_FONT_VECTOR
	VerFileFontTrueType           VerFileSubtype = wrappers.VFT2_FONT_TRUETYPE
)

type FileVersionNumber struct {
	Major    uint
	Minor    uint
	Build    uint
	Revision uint
}

func (self *FileVersionNumber) String() string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		self.Major,
		self.Minor,
		self.Build,
		self.Revision)
}

func StringToFileVersionNumber(s string) (FileVersionNumber, error) {
	var version FileVersionNumber
	parts := strings.Split(s, ".")
	if len(parts) >= 1 {
		n, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return FileVersionNumber{}, err
		}
		version.Major = uint(n)
	}
	if len(parts) >= 2 {
		n, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return FileVersionNumber{}, err
		}
		version.Minor = uint(n)
	}
	if len(parts) >= 3 {
		n, err := strconv.ParseUint(parts[2], 10, 16)
		if err != nil {
			return FileVersionNumber{}, err
		}
		version.Build = uint(n)
	}
	if len(parts) >= 4 {
		n, err := strconv.ParseUint(parts[3], 10, 16)
		if err != nil {
			return FileVersionNumber{}, err
		}
		version.Revision = uint(n)
	}
	return version, nil
}

func CompareFileVersionNumbers(v1, v2 FileVersionNumber) int {
	if v1.Major < v2.Major {
		return -1
	} else if v1.Major > v2.Major {
		return 1
	} else if v1.Minor < v2.Minor {
		return -1
	} else if v1.Minor > v2.Minor {
		return 1
	} else if v1.Build < v2.Build {
		return -1
	} else if v1.Build > v2.Build {
		return 1
	} else if v1.Revision < v2.Revision {
		return -1
	} else if v1.Revision > v2.Revision {
		return 1
	} else {
		return 0
	}
}

type FixedFileInfo struct {
	FileVersion    FileVersionNumber
	ProductVersion FileVersionNumber
	FileFlags      VerFileFlags
	FileOS         VerFileOS
	FileType       VerFileType
	FileSubtype    VerFileSubtype
}

type FileVersion struct {
	data []byte
}

func GetFileVersion(filename string) (*FileVersion, error) {
	var handle uint32
	size, err := wrappers.GetFileVersionInfoSize(syscall.StringToUTF16Ptr(filename), &handle)
	if err != nil {
		return nil, NewWindowsError("GetFileVersionInfoSize", err)
	}
	data := make([]byte, size)
	if err := wrappers.GetFileVersionInfo(syscall.StringToUTF16Ptr(filename), handle, size, &data[0]); err != nil {
		return nil, NewWindowsError("GetFileVersionInfo", err)
	}
	return &FileVersion{data: data}, nil
}

func (self *FileVersion) GetFixedFileInfo() (*FixedFileInfo, error) {
	var ffi *wrappers.VS_FIXEDFILEINFO
	var len uint32
	err := wrappers.VerQueryValue(
		&self.data[0],
		syscall.StringToUTF16Ptr("\\"),
		(**byte)(unsafe.Pointer(&ffi)),
		&len)
	if err != nil {
		return nil, NewWindowsError("VerQueryValue", err)
	}
	return &FixedFileInfo{
		FileVersion: FileVersionNumber{
			Major:    uint(wrappers.HIWORD(ffi.FileVersionMS)),
			Minor:    uint(wrappers.LOWORD(ffi.FileVersionMS)),
			Build:    uint(wrappers.HIWORD(ffi.FileVersionLS)),
			Revision: uint(wrappers.LOWORD(ffi.FileVersionLS)),
		},

		ProductVersion: FileVersionNumber{
			Major:    uint(wrappers.HIWORD(ffi.ProductVersionMS)),
			Minor:    uint(wrappers.LOWORD(ffi.ProductVersionMS)),
			Build:    uint(wrappers.HIWORD(ffi.ProductVersionLS)),
			Revision: uint(wrappers.LOWORD(ffi.ProductVersionLS)),
		},

		FileFlags:   VerFileFlags(ffi.FileFlags),
		FileOS:      VerFileOS(ffi.FileOS),
		FileType:    VerFileType(ffi.FileType),
		FileSubtype: VerFileSubtype(ffi.FileSubtype),
	}, nil
}
