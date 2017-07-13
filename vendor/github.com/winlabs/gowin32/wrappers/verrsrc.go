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

package wrappers

const (
	VS_FF_DEBUG        = 0x00000001
	VS_FF_PRERELEASE   = 0x00000002
	VS_FF_PATCHED      = 0x00000004
	VS_FF_PRIVATEBUILD = 0x00000008
	VS_FF_INFOINFERRED = 0x00000010
	VS_FF_SPECIALBUILD = 0x00000020
)

const (
	VOS_UNKNOWN       = 0x00000000
	VOS_DOS           = 0x00010000
	VOS_OS216         = 0x00020000
	VOS_OS232         = 0x00030000
	VOS_NT            = 0x00040000
	VOS__WINDOWS16    = 0x00000001
	VOS__PM16         = 0x00000002
	VOS__PM32         = 0x00000003
	VOS__WINDOWS32    = 0x00000004
	VOS_DOS_WINDOWS16 = 0x00010001
	VOS_DOS_WINDOWS32 = 0x00010004
	VOS_OS216_PM16    = 0x00020002
	VOS_OS232_PM32    = 0x00030003
	VOS_NT_WINDOWS32  = 0x00040004
)

const (
	VFT_UNKNOWN    = 0x00000000
	VFT_APP        = 0x00000001
	VFT_DLL        = 0x00000002
	VFT_DRV        = 0x00000003
	VFT_FONT       = 0x00000004
	VFT_VXD        = 0x00000005
	VFT_STATIC_LIB = 0x00000007
)

const (
	VFT2_UNKNOWN               = 0x00000000
	VFT2_DRV_PRINTER           = 0x00000001
	VFT2_DRV_KEYBOARD          = 0x00000002
	VFT2_DRV_LANGUAGE          = 0x00000003
	VFT2_DRV_DISPLAY           = 0x00000004
	VFT2_DRV_MOUSE             = 0x00000005
	VFT2_DRV_NETWORK           = 0x00000006
	VFT2_DRV_SYSTEM            = 0x00000007
	VFT2_DRV_INSTALLABLE       = 0x00000008
	VFT2_DRV_SOUND             = 0x00000009
	VFT2_DRV_COMM              = 0x0000000A
	VFT2_DRV_VERSIONED_PRINTER = 0x0000000C
	VFT2_FONT_RASTER           = 0x00000001
	VFT2_FONT_VECTOR           = 0x00000002
	VFT2_FONT_TRUETYPE         = 0x00000003
)

type VS_FIXEDFILEINFO struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}
