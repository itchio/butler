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

func MAKEINTRESOURCE(integer uint16) uintptr {
	return uintptr(integer)
}

var (
	RT_CURSOR       = MAKEINTRESOURCE(1)
	RT_BITMAP       = MAKEINTRESOURCE(2)
	RT_ICON         = MAKEINTRESOURCE(3)
	RT_MENU         = MAKEINTRESOURCE(4)
	RT_DIALOG       = MAKEINTRESOURCE(5)
	RT_STRING       = MAKEINTRESOURCE(6)
	RT_FONTDIR      = MAKEINTRESOURCE(7)
	RT_FONT         = MAKEINTRESOURCE(8)
	RT_ACCELERATOR  = MAKEINTRESOURCE(9)
	RT_RCDATA       = MAKEINTRESOURCE(10)
	RT_MESSAGETABLE = MAKEINTRESOURCE(11)
	RT_GROUP_CURSOR = MAKEINTRESOURCE(12)
	RT_GROUP_ICON   = MAKEINTRESOURCE(14)
	RT_VERSION      = MAKEINTRESOURCE(16)
	RT_DLGINCLUDE   = MAKEINTRESOURCE(17)
	RT_PLUGPLAY     = MAKEINTRESOURCE(19)
	RT_VXD          = MAKEINTRESOURCE(20)
	RT_ANICURSOR    = MAKEINTRESOURCE(21)
	RT_ANIICON      = MAKEINTRESOURCE(22)
	RT_HTML         = MAKEINTRESOURCE(23)
	RT_MANIFEST     = MAKEINTRESOURCE(24)
)

var (
	CREATEPROCESS_MANIFEST_RESOURCE_ID                 = MAKEINTRESOURCE(1)
	ISOLATIONAWARE_MANIFEST_RESOURCE_ID                = MAKEINTRESOURCE(2)
	ISOLATIONAWARE_NOSTATICIMPORT_MANIFEST_RESOURCE_ID = MAKEINTRESOURCE(3)
)

const (
	SW_HIDE            = 0
	SW_SHOWNORMAL      = 1
	SW_SHOWMINIMIZED   = 2
	SW_SHOWMAXIMIZED   = 3
	SW_MAXIMIZE        = 3
	SW_SHOWNOACTIVATE  = 4
	SW_SHOW            = 5
	SW_MINIMIZE        = 6
	SW_SHOWMINNOACTIVE = 7
	SW_SHOWNA          = 8
	SW_RESTORE         = 9
	SW_SHOWDEFAULT     = 10
	SW_FORCEMINIMIZE   = 11
)
