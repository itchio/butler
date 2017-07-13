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
	FILE_DEVICE_BEEP                = 0x00000001
	FILE_DEVICE_CD_ROM              = 0x00000002
	FILE_DEVICE_CD_ROM_FILE_SYSTEM  = 0x00000003
	FILE_DEVICE_CONTROLLER          = 0x00000004
	FILE_DEVICE_DATALINK            = 0x00000005
	FILE_DEVICE_DFS                 = 0x00000006
	FILE_DEVICE_DISK                = 0x00000007
	FILE_DEVICE_DISK_FILE_SYSTEM    = 0x00000008
	FILE_DEVICE_FILE_SYSTEM         = 0x00000009
	FILE_DEVICE_INPUT_PORT          = 0x0000000A
	FILE_DEVICE_KEYBOARD            = 0x0000000B
	FILE_DEVICE_MAILSLOT            = 0x0000000C
	FILE_DEVICE_MIDI_IN             = 0x0000000D
	FILE_DEVICE_MIDI_OUT            = 0x0000000E
	FILE_DEVICE_MOUSE               = 0x0000000F
	FILE_DEVICE_MULTI_UNC_PROVIDER  = 0x00000010
	FILE_DEVICE_NAMED_PIPE          = 0x00000011
	FILE_DEVICE_NETWORK             = 0x00000012
	FILE_DEVICE_NETWORK_BROWSER     = 0x00000013
	FILE_DEVICE_NETWORK_FILE_SYSTEM = 0x00000014
	FILE_DEVICE_NULL                = 0x00000015
	FILE_DEVICE_PARALLEL_PORT       = 0x00000016
	FILE_DEVICE_PHYSICAL_NETCARD    = 0x00000017
	FILE_DEVICE_PRINTER             = 0x00000018
	FILE_DEVICE_SCANNER             = 0x00000019
	FILE_DEVICE_SERIAL_MOUSE_PORT   = 0x0000001A
	FILE_DEVICE_SERIAL_PORT         = 0x0000001B
	FILE_DEVICE_SCREEN              = 0x0000001C
	FILE_DEVICE_SOUND               = 0x0000001D
	FILE_DEVICE_STREAMS             = 0x0000001E
	FILE_DEVICE_TAPE                = 0x0000001F
	FILE_DEVICE_TAPE_FILE_SYSTEM    = 0x00000020
	FILE_DEVICE_TRANSPORT           = 0x00000021
	FILE_DEVICE_UNKNOWN             = 0x00000022
	FILE_DEVICE_VIDEO               = 0x00000023
	FILE_DEVICE_VIRTUAL_DISK        = 0x00000024
	FILE_DEVICE_WAVE_IN             = 0x00000025
	FILE_DEVICE_WAVE_OUT            = 0x00000026
	FILE_DEVICE_8042_PORT           = 0x00000027
	FILE_DEVICE_NETWORK_REDIRECTOR  = 0x00000028
	FILE_DEVICE_BATTERY             = 0x00000029
	FILE_DEVICE_BUS_EXTENDER        = 0x0000002A
	FILE_DEVICE_MODEM               = 0x0000002B
	FILE_DEVICE_VDM                 = 0x0000002C
	FILE_DEVICE_MASS_STORAGE        = 0x0000002D
	FILE_DEVICE_SMB                 = 0x0000002E
	FILE_DEVICE_KS                  = 0x0000002F
	FILE_DEVICE_CHANGER             = 0x00000030
	FILE_DEVICE_SMARTCARD           = 0x00000031
	FILE_DEVICE_ACPI                = 0x00000032
	FILE_DEVICE_DVD                 = 0x00000033
	FILE_DEVICE_FULLSCREEN_VIDEO    = 0x00000034
	FILE_DEVICE_DFS_FILE_SYSTEM     = 0x00000035
	FILE_DEVICE_DFS_VOLUME          = 0x00000036
	FILE_DEVICE_SERENUM             = 0x00000037
	FILE_DEVICE_TERMSRV             = 0x00000038
	FILE_DEVICE_KSEC                = 0x00000039
	FILE_DEVICE_FIPS                = 0x0000003A
)

func CTL_CODE(deviceType uint32, function uint32, method uint32, access uint32) uint32 {
	return (deviceType << 16) | (access << 14) | (function << 2) | method
}

const (
	METHOD_BUFFERED   = 0
	METHOD_IN_DIRECT  = 1
	METHOD_OUT_DIRECT = 2
	METHOD_NEITHER    = 3
)

const (
	FILE_ANY_ACCESS     = 0x0000
	FILE_SPECIAL_ACCESS = FILE_ANY_ACCESS
	FILE_READ_ACCESS    = 0x0001
	FILE_WRITE_ACCESS   = 0x0002
)

var (
	IOCTL_DISK_PERFORMANCE = CTL_CODE(FILE_DEVICE_DISK, 0x0008, METHOD_BUFFERED, FILE_ANY_ACCESS)
)

var (
	FSCTL_SET_REPARSE_POINT    = CTL_CODE(FILE_DEVICE_FILE_SYSTEM, 41, METHOD_BUFFERED, FILE_SPECIAL_ACCESS)
	FSCTL_GET_REPARSE_POINT    = CTL_CODE(FILE_DEVICE_FILE_SYSTEM, 42, METHOD_BUFFERED, FILE_ANY_ACCESS)
	FSCTL_DELETE_REPARSE_POINT = CTL_CODE(FILE_DEVICE_FILE_SYSTEM, 42, METHOD_BUFFERED, FILE_SPECIAL_ACCESS)
)

type DISK_PERFORMANCE struct {
	BytesRead           int64
	BytesWritten        int64
	ReadTime            int64
	WriteTime           int64
	IdleTime            int64
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           int64
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
}
