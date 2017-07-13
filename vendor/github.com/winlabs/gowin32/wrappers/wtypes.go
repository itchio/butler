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
	CLSCTX_INPROC_SERVER          = 0x00000001
	CLSCTX_INPROC_HANDLER         = 0x00000002
	CLSCTX_LOCAL_SERVER           = 0x00000004
	CLSCTX_INPROC_SERVER16        = 0x00000008
	CLSCTX_REMOTE_SERVER          = 0x00000010
	CLSCTX_INPROC_HANDLER16       = 0x00000020
	CLSCTX_RESERVED1              = 0x00000040
	CLSCTX_RESERVED2              = 0x00000080
	CLSCTX_RESERVED3              = 0x00000100
	CLSCTX_RESERVED4              = 0x00000200
	CLSCTX_NO_CODE_DOWNLOAD       = 0x00000400
	CLSCTX_RESERVED5              = 0x00000800
	CLSCTX_NO_CUSTOM_MARSHAL      = 0x00001000
	CLSCTX_ENABLE_CODE_DOWNLOAD   = 0x00002000
	CLSCTX_NO_FAILURE_LOG         = 0x00004000
	CLSCTX_DISABLE_AAA            = 0x00008000
	CLSCTX_ENABLE_AAA             = 0x00010000
	CLSCTX_FROM_DEFAULT_CONTEXT   = 0x00020000
	CLSCTX_ACTIVATE_32_BIT_SERVER = 0x00040000
	CLSCTX_ACTIVATE_64_BIT_SERVER = 0x00080000
	CLSCTX_ENABLE_CLOAKING        = 0x00100000
	CLSCTX_PS_DLL                 = 0x80000000
)

const (
	VARIANT_TRUE  = -1
	VARIANT_FALSE = 0
)

const (
	VT_EMPTY            = 0
	VT_NULL             = 1
	VT_I2               = 2
	VT_I4               = 3
	VT_R4               = 4
	VT_R8               = 5
	VT_CY               = 6
	VT_DATE             = 7
	VT_BSTR             = 8
	VT_DISPATCH         = 9
	VT_ERROR            = 10
	VT_BOOL             = 11
	VT_VARIANT          = 12
	VT_UNKNOWN          = 13
	VT_DECIMAL          = 14
	VT_I1               = 16
	VT_UI1              = 17
	VT_UI2              = 18
	VT_UI4              = 19
	VT_I8               = 20
	VT_UI8              = 21
	VT_INT              = 22
	VT_UINT             = 23
	VT_VOID             = 24
	VT_HRESULT          = 25
	VT_PTR              = 26
	VT_SAFEARRAY        = 27
	VT_CARRAY           = 28
	VT_USERDEFINED      = 29
	VT_LPSTR            = 30
	VT_LPWSTR           = 31
	VT_RECORD           = 36
	VT_INT_PTR          = 37
	VT_UINT_PTR         = 38
	VT_FILETIME         = 64
	VT_BLOB             = 65
	VT_STREAM           = 66
	VT_STORAGE          = 67
	VT_STREAMED_OBJECT  = 68
	VT_STORED_OBJECT    = 69
	VT_BLOB_OBJECT      = 70
	VT_CF               = 71
	VT_CLSID            = 72
	VT_VERSIONED_STREAM = 73
	VT_BSTR_BLOB        = 0x0FFF
	VT_VECTOR           = 0x1000
	VT_ARRAY            = 0x2000
	VT_BYREF            = 0x4000
)
