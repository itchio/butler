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

type COMInitFlags uint32

const (
	COMInitApartmentThreaded COMInitFlags = wrappers.COINIT_APARTMENTTHREADED
	COMInitMultithreaded     COMInitFlags = wrappers.COINIT_MULTITHREADED
	COMInitDisableOLE1DDE    COMInitFlags = wrappers.COINIT_DISABLE_OLE1DDE
	COMInitSpeedOverMemory   COMInitFlags = wrappers.COINIT_SPEED_OVER_MEMORY
)

func InitializeCOM(flags COMInitFlags) error {
	if hr := wrappers.CoInitializeEx(nil, uint32(flags)); wrappers.FAILED(hr) {
		return NewWindowsError("CoInitializeEx", COMError(hr))
	}
	return nil
}

func UninitializeCOM() {
	wrappers.CoUninitialize()
}
