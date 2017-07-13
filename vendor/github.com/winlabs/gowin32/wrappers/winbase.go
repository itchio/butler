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

import (
	"syscall"
	"unsafe"
)

const (
	INVALID_HANDLE_VALUE    = ^syscall.Handle(0)
	INVALID_FILE_SIZE       = 0xFFFFFFFF
	INVALID_FILE_ATTRIBUTES = 0xFFFFFFFF
)

const (
	WAIT_FAILED        = 0xFFFFFFFF
	WAIT_OBJECT_0      = STATUS_WAIT_0
	WAIT_ABANDONED     = STATUS_ABANDONED_WAIT_0
	WAIT_ABANDONED_0   = STATUS_ABANDONED_WAIT_0
	WAIT_IO_COMPLETION = STATUS_USER_APC
)

const (
	FILE_FLAG_WRITE_THROUGH       = 0x80000000
	FILE_FLAG_OVERLAPPED          = 0x40000000
	FILE_FLAG_NO_BUFFERING        = 0x20000000
	FILE_FLAG_RANDOM_ACCESS       = 0x10000000
	FILE_FLAG_SEQUENTIAL_SCAN     = 0x08000000
	FILE_FLAG_DELETE_ON_CLOSE     = 0x04000000
	FILE_FLAG_BACKUP_SEMANTICS    = 0x02000000
	FILE_FLAG_POSIX_SEMANTICS     = 0x01000000
	FILE_FLAG_OPEN_REPARSE_POINT  = 0x00200000
	FILE_FLAG_OPEN_NO_RECALL      = 0x00100000
	FILE_FLAG_FIRST_PIPE_INSTANCE = 0x00080000
)

const (
	CREATE_NEW        = 1
	CREATE_ALWAYS     = 2
	OPEN_EXISTING     = 3
	OPEN_ALWAYS       = 4
	TRUNCATE_EXISTING = 5
)

const (
	SECURITY_ANONYMOUS        = SecurityAnonymous << 16
	SECURITY_IDENTIFICATION   = SecurityIdentification << 16
	SECURITY_IMPERSONATION    = SecurityImpersonation << 16
	SECURITY_DELEGATION       = SecurityDelegation << 16
	SECURITY_CONTEXT_TRACKING = 0x00040000
	SECURITY_EFFECTIVE_ONLY   = 0x00080000
)

type OVERLAPPED struct {
	Internal     uintptr
	InternalHigh uintptr
	Offset       uint32
	OffsetHigh   uint32
	Event        syscall.Handle
}

type SECURITY_ATTRIBUTES struct {
	Length             uint32
	SecurityDescriptor *byte
	InheritHandle      int32
}

type PROCESS_INFORMATION struct {
	Process   syscall.Handle
	Thread    syscall.Handle
	ProcessId uint32
	ThreadId  uint32
}

type FILETIME struct {
	LowDateTime  uint32
	HighDateTime uint32
}

type CRITICAL_SECTION RTL_CRITICAL_SECTION

type SYSTEM_INFO struct {
	ProcessorArchitecture     uint16
	Reserved                  uint16
	PageSize                  uint32
	MinimumApplicationAddress *byte
	MaximumApplicationAddress *byte
	ActiveProcessorMask       uintptr
	NumberOfProcessors        uint32
	ProcessorType             uint32
	AllocationGranularity     uint32
	ProcessorLevel            uint16
	ProcessorRevision         uint16
}

const (
	DEBUG_PROCESS                    = 0x00000001
	DEBUG_ONLY_THIS_PROCESS          = 0x00000002
	CREATE_SUSPENDED                 = 0x00000004
	DETACHED_PROCESS                 = 0x00000008
	CREATE_NEW_CONSOLE               = 0x00000010
	NORMAL_PRIORITY_CLASS            = 0x00000020
	IDLE_PRIORITY_CLASS              = 0x00000040
	HIGH_PRIORITY_CLASS              = 0x00000080
	REALTIME_PRIORITY_CLASS          = 0x00000100
	CREATE_NEW_PROCESS_GROUP         = 0x00000200
	CREATE_UNICODE_ENVIRONMENT       = 0x00000400
	CREATE_SEPARATE_WOW_VDM          = 0x00000800
	CREATE_SHARED_WOW_VDM            = 0x00001000
	BELOW_NORMAL_PRIORITY_CLASS      = 0x00004000
	ABOVE_NORMAL_PRIORITY_CLASS      = 0x00008000
	INHERIT_PARENT_AFFINITY          = 0x00010000
	CREATE_PROTECTED_PROCESS         = 0x00040000
	EXTENDED_STARTUPINFO_PRESENT     = 0x00080000
	PROCESS_MODE_BACKGROUND_BEGIN    = 0x00100000
	PROCESS_MODE_BACKGROUND_END      = 0x00200000
	CREATE_BREAKAWAY_FROM_JOB        = 0x01000000
	CREATE_PRESERVE_CODE_AUTHZ_LEVEL = 0x02000000
	CREATE_DEFAULT_ERROR_MODE        = 0x04000000
	CREATE_NO_WINDOW                 = 0x08000000
)

const (
	DRIVE_UNKNOWN     = 0
	DRIVE_NO_ROOT_DIR = 1
	DRIVE_REMOVABLE   = 2
	DRIVE_FIXED       = 3
	DRIVE_REMOTE      = 4
	DRIVE_CDROM       = 5
	DRIVE_RAMDISK     = 6
)

const (
	STD_INPUT_HANDLE  = ^uint32(10) + 1
	STD_OUTPUT_HANDLE = ^uint32(11) + 1
	STD_ERROR_HANDLE  = ^uint32(12) + 1
)

const (
	INFINITE = 0xFFFFFFFF
)

const (
	FORMAT_MESSAGE_ALLOCATE_BUFFER = 0x00000100
	FORMAT_MESSAGE_IGNORE_INSERTS  = 0x00000200
	FORMAT_MESSAGE_FROM_STRING     = 0x00000400
	FORMAT_MESSAGE_FROM_HMODULE    = 0x00000800
	FORMAT_MESSAGE_FROM_SYSTEM     = 0x00001000
	FORMAT_MESSAGE_ARGUMENT_ARRAY  = 0x00002000
	FORMAT_MESSAGE_MAX_WIDTH_MASK  = 0x000000FF
)

const (
	STARTF_USESHOWWINDOW    = 0x00000001
	STARTF_USESIZE          = 0x00000002
	STARTF_USEPOSITION      = 0x00000004
	STARTF_USECOUNTCHARS    = 0x00000008
	STARTF_USEFILLATTRIBUTE = 0x00000010
	STARTF_RUNFULLSCREEN    = 0x00000020
	STARTF_FORCEONFEEDBACK  = 0x00000040
	STARTF_FORCEOFFFEEDBACK = 0x00000080
	STARTF_USESTDHANDLES    = 0x00000100
	STARTF_USEHOTKEY        = 0x00000200
	STARTF_TITLEISLINKNAME  = 0x00000800
	STARTF_TITLEISAPPID     = 0x00001000
	STARTF_PREVENTPINNING   = 0x00002000
)

type STARTUPINFO struct {
	Cb            uint32
	Reserved      *uint16
	Desktop       *uint16
	Title         *uint16
	X             uint32
	Y             uint32
	XSize         uint32
	YSize         uint32
	XCountChars   uint32
	YCountChars   uint32
	FillAttribute uint32
	Flags         uint32
	ShowWindow    uint16
	CbReserved2   uint16
	Reserved2     *byte
	StdInput      syscall.Handle
	StdOutput     syscall.Handle
	StdError      syscall.Handle
}

type WIN32_FIND_DATA struct {
	FileAttributes    uint32
	CreationTime      FILETIME
	LastAccessTime    FILETIME
	LastWriteTime     FILETIME
	FileSizeHigh      uint32
	FileSizeLow       uint32
	Reserved0         uint32
	Reserved1         uint32
	FileName          [MAX_PATH]uint16
	AlternateFileName [14]uint16
}

const (
	PROCESS_NAME_NATIVE = 0x00000001
)

const (
	MOVEFILE_REPLACE_EXISTING      = 0x00000001
	MOVEFILE_COPY_ALLOWED          = 0x00000002
	MOVEFILE_DELAY_UNTIL_REBOOT    = 0x00000004
	MOVEFILE_WRITE_THROUGH         = 0x00000008
	MOVEFILE_CREATE_HARDLINK       = 0x00000010
	MOVEFILE_FAIL_IF_NOT_TRACKABLE = 0x00000020
)

const (
	MAX_COMPUTERNAME_LENGTH = 15
)

const (
	ComputerNameNetBIOS                   = 0
	ComputerNameDnsHostname               = 1
	ComputerNameDnsDomain                 = 2
	ComputerNameDnsFullyQualified         = 3
	ComputerNamePhysicalNetBIOS           = 4
	ComputerNamePhysicalDnsHostname       = 5
	ComputerNamePhysicalDnsDomain         = 6
	ComputerNamePhysicalDnsFullyQualified = 7
)

const (
	SYMBOLIC_LINK_FLAG_DIRECTORY = 0x00000001
)

var (
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modadvapi32 = syscall.NewLazyDLL("advapi32.dll")

	procAssignProcessToJobObject          = modkernel32.NewProc("AssignProcessToJobObject")
	procBeginUpdateResourceW              = modkernel32.NewProc("BeginUpdateResourceW")
	procCloseHandle                       = modkernel32.NewProc("CloseHandle")
	procCopyFileW                         = modkernel32.NewProc("CopyFileW")
	procCreateFileW                       = modkernel32.NewProc("CreateFileW")
	procCreateJobObjectW                  = modkernel32.NewProc("CreateJobObjectW")
	procCreateProcessW                    = modkernel32.NewProc("CreateProcessW")
	procCreateSymbolicLinkW               = modkernel32.NewProc("CreateSymbolicLinkW")
	procDeleteCriticalSection             = modkernel32.NewProc("DeleteCriticalSection")
	procDeleteFileW                       = modkernel32.NewProc("DeleteFileW")
	procDeviceIoControl                   = modkernel32.NewProc("DeviceIoControl")
	procEndUpdateResourceW                = modkernel32.NewProc("EndUpdateResourceW")
	procEnterCriticalSection              = modkernel32.NewProc("EnterCriticalSection")
	procExpandEnvironmentStringsW         = modkernel32.NewProc("ExpandEnvironmentStringsW")
	procFindClose                         = modkernel32.NewProc("FindClose")
	procFindFirstFileW                    = modkernel32.NewProc("FindFirstFileW")
	procFindNextFileW                     = modkernel32.NewProc("FindNextFileW")
	procFormatMessageW                    = modkernel32.NewProc("FormatMessageW")
	procFreeEnvironmentStringsW           = modkernel32.NewProc("FreeEnvironmentStringsW")
	procFreeLibrary                       = modkernel32.NewProc("FreeLibrary")
	procGetCompressedFileSizeW            = modkernel32.NewProc("GetCompressedFileSizeW")
	procGetComputerNameExW                = modkernel32.NewProc("GetComputerNameExW")
	procGetComputerNameW                  = modkernel32.NewProc("GetComputerNameW")
	procGetCurrentProcess                 = modkernel32.NewProc("GetCurrentProcess")
	procGetCurrentThread                  = modkernel32.NewProc("GetCurrentThread")
	procGetDriveTypeW                     = modkernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceExW               = modkernel32.NewProc("GetDiskFreeSpaceExW")
	procGetDiskFreeSpaceW                 = modkernel32.NewProc("GetDiskFreeSpaceW")
	procGetEnvironmentStringsW            = modkernel32.NewProc("GetEnvironmentStringsW")
	procGetEnvironmentVariableW           = modkernel32.NewProc("GetEnvironmentVariableW")
	procGetFileAttributesW                = modkernel32.NewProc("GetFileAttributesW")
	procGetFileSize                       = modkernel32.NewProc("GetFileSize")
	procGetModuleFileNameW                = modkernel32.NewProc("GetModuleFileNameW")
	procGetProcessTimes                   = modkernel32.NewProc("GetProcessTimes")
	procGetStdHandle                      = modkernel32.NewProc("GetStdHandle")
	procGetSystemDirectoryW               = modkernel32.NewProc("GetSystemDirectoryW")
	procGetSystemInfo                     = modkernel32.NewProc("GetSystemInfo")
	procGetSystemTimeAsFileTime           = modkernel32.NewProc("GetSystemTimeAsFileTime")
	procGetSystemTimes                    = modkernel32.NewProc("GetSystemTimes")
	procGetSystemWindowsDirectoryW        = modkernel32.NewProc("GetSystemWindowsDirectoryW")
	procGetSystemWow64DirectoryW          = modkernel32.NewProc("GetSystemWow64DirectoryW")
	procGetTempFileNameW                  = modkernel32.NewProc("GetTempFileNameW")
	procGetTempPathW                      = modkernel32.NewProc("GetTempPathW")
	procGetVersionExW                     = modkernel32.NewProc("GetVersionExW")
	procGetVolumeInformationW             = modkernel32.NewProc("GetVolumeInformationW")
	procGetVolumeNameForVolumeMountPointW = modkernel32.NewProc("GetVolumeNameForVolumeMountPointW")
	procGetVolumePathNameW                = modkernel32.NewProc("GetVolumePathNameW")
	procGetWindowsDirectoryW              = modkernel32.NewProc("GetWindowsDirectoryW")
	procInitializeCriticalSection         = modkernel32.NewProc("InitializeCriticalSection")
	procIsProcessInJob                    = modkernel32.NewProc("IsProcessInJob")
	procLeaveCriticalSection              = modkernel32.NewProc("LeaveCriticalSection")
	procLoadLibraryW                      = modkernel32.NewProc("LoadLibraryW")
	procLocalFree                         = modkernel32.NewProc("LocalFree")
	procMoveFileExW                       = modkernel32.NewProc("MoveFileExW")
	procMoveFileW                         = modkernel32.NewProc("MoveFileW")
	procOpenJobObjectW                    = modkernel32.NewProc("OpenJobObjectW")
	procOpenProcess                       = modkernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW        = modkernel32.NewProc("QueryFullProcessImageNameW")
	procQueryInformationJobObject         = modkernel32.NewProc("QueryInformationJobObject")
	procReadFile                          = modkernel32.NewProc("ReadFile")
	procReadProcessMemory                 = modkernel32.NewProc("ReadProcessMemory")
	procSetEnvironmentVariableW           = modkernel32.NewProc("SetEnvironmentVariableW")
	procSetFileAttributesW                = modkernel32.NewProc("SetFileAttributesW")
	procSetFileTime                       = modkernel32.NewProc("SetFileTime")
	procSetInformationJobObject           = modkernel32.NewProc("SetInformationJobObject")
	procSetStdHandle                      = modkernel32.NewProc("SetStdHandle")
	procTerminateJobObject                = modkernel32.NewProc("TerminateJobObject")
	procTerminateProcess                  = modkernel32.NewProc("TerminateProcess")
	procTryEnterCriticalSection           = modkernel32.NewProc("TryEnterCriticalSection")
	procUpdateResourceW                   = modkernel32.NewProc("UpdateResourceW")
	procVerifyVersionInfoW                = modkernel32.NewProc("VerifyVersionInfoW")
	procWaitForSingleObject               = modkernel32.NewProc("WaitForSingleObject")
	proclstrlenW                          = modkernel32.NewProc("lstrlenW")

	procAdjustTokenPrivileges      = modadvapi32.NewProc("AdjustTokenPrivileges")
	procAllocateAndInitializeSid   = modadvapi32.NewProc("AllocateAndInitializeSid")
	procCheckTokenMembership       = modadvapi32.NewProc("CheckTokenMembership")
	procCopySid                    = modadvapi32.NewProc("CopySid")
	procDeregisterEventSource      = modadvapi32.NewProc("DeregisterEventSource")
	procEqualSid                   = modadvapi32.NewProc("EqualSid")
	procFreeSid                    = modadvapi32.NewProc("FreeSid")
	procGetFileSecurityW           = modadvapi32.NewProc("GetFileSecurityW")
	procGetLengthSid               = modadvapi32.NewProc("GetLengthSid")
	procGetSecurityDescriptorOwner = modadvapi32.NewProc("GetSecurityDescriptorOwner")
	procGetTokenInformation        = modadvapi32.NewProc("GetTokenInformation")
	procImpersonateSelf            = modadvapi32.NewProc("ImpersonateSelf")
	procLookupAccountNameW         = modadvapi32.NewProc("LookupAccountNameW")
	procLookupPrivilegeValueW      = modadvapi32.NewProc("LookupPrivilegeValueW")
	procOpenProcessToken           = modadvapi32.NewProc("OpenProcessToken")
	procOpenThreadToken            = modadvapi32.NewProc("OpenThreadToken")
	procRegisterEventSourceW       = modadvapi32.NewProc("RegisterEventSourceW")
	procReportEventW               = modadvapi32.NewProc("ReportEventW")
	procRevertToSelf               = modadvapi32.NewProc("RevertToSelf")
)

func AssignProcessToJobObject(job syscall.Handle, process syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(
		procAssignProcessToJobObject.Addr(),
		2,
		uintptr(job),
		uintptr(process),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func BeginUpdateResource(fileName *uint16, deleteExistingResources bool) (syscall.Handle, error) {
	var deleteExistingResourcesRaw int32
	if deleteExistingResources {
		deleteExistingResourcesRaw = 1
	} else {
		deleteExistingResourcesRaw = 0
	}
	r1, _, e1 := syscall.Syscall(
		procBeginUpdateResourceW.Addr(),
		2,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(deleteExistingResourcesRaw),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func CloseHandle(object syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procCloseHandle.Addr(), 1, uintptr(object), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CopyFile(existingFileName *uint16, newFileName *uint16, failIfExists bool) error {
	var failIfExistsRaw int32
	if failIfExists {
		failIfExistsRaw = 1
	} else {
		failIfExistsRaw = 0
	}
	r1, _, e1 := syscall.Syscall(
		procCopyFileW.Addr(),
		3,
		uintptr(unsafe.Pointer(existingFileName)),
		uintptr(unsafe.Pointer(newFileName)),
		uintptr(failIfExistsRaw))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CreateFile(fileName *uint16, desiredAccess uint32, shareMode uint32, securityAttributes *SECURITY_ATTRIBUTES, creationDisposition uint32, flagsAndAttributes uint32, templateFile syscall.Handle) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall9(
		procCreateFileW.Addr(),
		7,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(desiredAccess),
		uintptr(shareMode),
		uintptr(unsafe.Pointer(securityAttributes)),
		uintptr(creationDisposition),
		uintptr(flagsAndAttributes),
		uintptr(templateFile),
		0,
		0)
	handle := syscall.Handle(r1)
	if handle == INVALID_HANDLE_VALUE {
		if e1 != ERROR_SUCCESS {
			return handle, e1
		} else {
			return handle, syscall.EINVAL
		}
	}
	return handle, nil
}

func CreateJobObject(jobAttributes *SECURITY_ATTRIBUTES, name *uint16) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procCreateJobObjectW.Addr(),
		2,
		uintptr(unsafe.Pointer(jobAttributes)),
		uintptr(unsafe.Pointer(name)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func CreateProcess(applicationName *uint16, commandLine *uint16, processAttributes *SECURITY_ATTRIBUTES, threadAttributes *SECURITY_ATTRIBUTES, inheritHandles bool, creationFlags uint32, environment *byte, currentDirectory *uint16, startupInfo *STARTUPINFO, processInformation *PROCESS_INFORMATION) error {
	var inheritHandlesRaw int32
	if inheritHandles {
		inheritHandlesRaw = 1
	} else {
		inheritHandlesRaw = 0
	}
	r1, _, e1 := syscall.Syscall12(
		procCreateProcessW.Addr(),
		10,
		uintptr(unsafe.Pointer(applicationName)),
		uintptr(unsafe.Pointer(commandLine)),
		uintptr(unsafe.Pointer(processAttributes)),
		uintptr(unsafe.Pointer(threadAttributes)),
		uintptr(inheritHandlesRaw),
		uintptr(creationFlags),
		uintptr(unsafe.Pointer(environment)),
		uintptr(unsafe.Pointer(currentDirectory)),
		uintptr(unsafe.Pointer(startupInfo)),
		uintptr(unsafe.Pointer(processInformation)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CreateSymbolicLink(symlinkFileName *uint16, targetFileName *uint16, flags uint32) error {
	r1, _, e1 := syscall.Syscall(
		procCreateSymbolicLinkW.Addr(),
		3,
		uintptr(unsafe.Pointer(symlinkFileName)),
		uintptr(unsafe.Pointer(targetFileName)),
		uintptr(flags))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func DeleteCriticalSection(criticalSection *CRITICAL_SECTION) {
	syscall.Syscall(procDeleteCriticalSection.Addr(), 1, uintptr(unsafe.Pointer(criticalSection)), 0, 0)
}

func DeleteFile(fileName *uint16) error {
	r1, _, e1 := syscall.Syscall(procDeleteFileW.Addr(), 1, uintptr(unsafe.Pointer(fileName)), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func DeviceIoControl(device syscall.Handle, ioControlCode uint32, inBuffer *byte, inBufferSize uint32, outBuffer *byte, outBufferSize uint32, bytesReturned *uint32, overlapped *syscall.Overlapped) error {
	r1, _, e1 := syscall.Syscall9(
		procDeviceIoControl.Addr(),
		8,
		uintptr(device),
		uintptr(ioControlCode),
		uintptr(unsafe.Pointer(inBuffer)),
		uintptr(inBufferSize),
		uintptr(unsafe.Pointer(outBuffer)),
		uintptr(outBufferSize),
		uintptr(unsafe.Pointer(bytesReturned)),
		uintptr(unsafe.Pointer(overlapped)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func EndUpdateResource(update syscall.Handle, discard bool) error {
	var discardRaw int32
	if discard {
		discardRaw = 1
	} else {
		discardRaw = 0
	}
	r1, _, e1 := syscall.Syscall(
		procEndUpdateResourceW.Addr(),
		2,
		uintptr(update),
		uintptr(discardRaw),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func EnterCriticalSection(criticalSection *CRITICAL_SECTION) {
	syscall.Syscall(procEnterCriticalSection.Addr(), 1, uintptr(unsafe.Pointer(criticalSection)), 0, 0)
}

func ExpandEnvironmentStrings(src *uint16, dst *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procExpandEnvironmentStringsW.Addr(),
		3,
		uintptr(unsafe.Pointer(src)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(size))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func FindClose(findFile syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procFindClose.Addr(), 1, uintptr(findFile), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func FindFirstFile(fileName *uint16, findFileData *WIN32_FIND_DATA) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procFindFirstFileW.Addr(),
		2,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(unsafe.Pointer(findFileData)),
		0)
	handle := syscall.Handle(r1)
	if handle == INVALID_HANDLE_VALUE {
		if e1 != ERROR_SUCCESS {
			return handle, e1
		} else {
			return handle, syscall.EINVAL
		}
	}
	return handle, nil
}

func FindNextFile(findFile syscall.Handle, findFileData *WIN32_FIND_DATA) error {
	r1, _, e1 := syscall.Syscall(
		procFindNextFileW.Addr(),
		2,
		uintptr(findFile),
		uintptr(unsafe.Pointer(findFileData)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func FormatMessage(flags uint32, source uintptr, messageId uint32, languageId uint32, buffer *uint16, size uint32, arguments *byte) (uint32, error) {
	r1, _, e1 := syscall.Syscall9(
		procFormatMessageW.Addr(),
		7,
		uintptr(flags),
		source,
		uintptr(messageId),
		uintptr(languageId),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		uintptr(unsafe.Pointer(arguments)),
		0,
		0)
	if r1 == 0 {
		if e1 != 0 {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func FreeEnvironmentStrings(environmentBlock *uint16) error {
	r1, _, e1 := syscall.Syscall(
		procFreeEnvironmentStringsW.Addr(),
		1,
		uintptr(unsafe.Pointer(environmentBlock)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func FreeLibrary(module syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procFreeLibrary.Addr(), 1, uintptr(module), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetCompressedFileSize(fileName *uint16, fileSizeHigh *uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetCompressedFileSizeW.Addr(),
		2,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(unsafe.Pointer(fileSizeHigh)),
		0)
	if r1 == INVALID_FILE_SIZE {
		if e1 != ERROR_SUCCESS {
			return uint32(r1), e1
		} else {
			return uint32(r1), syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetComputerName(buffer *uint16, size *uint32) error {
	r1, _, e1 := syscall.Syscall(
		procGetComputerNameW.Addr(),
		2,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(size)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetComputerNameEx(nameType uint32, buffer *uint16, size *uint32) error {
	r1, _, e1 := syscall.Syscall(
		procGetComputerNameExW.Addr(),
		3,
		uintptr(nameType),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(unsafe.Pointer(size)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetCurrentProcess() syscall.Handle {
	r1, _, _ := syscall.Syscall(procGetCurrentProcess.Addr(), 0, 0, 0, 0)
	return syscall.Handle(r1)
}

func GetCurrentThread() syscall.Handle {
	r1, _, _ := syscall.Syscall(procGetCurrentThread.Addr(), 0, 0, 0, 0)
	return syscall.Handle(r1)
}

func GetDiskFreeSpace(rootPathName *uint16, sectorsPerCluster *uint32, bytesPerSector *uint32, numberOfFreeClusters *uint32, totalNumberOfClusters *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procGetDiskFreeSpaceW.Addr(),
		5,
		uintptr(unsafe.Pointer(rootPathName)),
		uintptr(unsafe.Pointer(sectorsPerCluster)),
		uintptr(unsafe.Pointer(bytesPerSector)),
		uintptr(unsafe.Pointer(numberOfFreeClusters)),
		uintptr(unsafe.Pointer(totalNumberOfClusters)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetDiskFreeSpaceEx(directoryName *uint16, freeBytesAvailable *uint64, totalNumberOfBytes *uint64, totalNumberOfFreeBytes *uint64) error {
	r1, _, e1 := syscall.Syscall6(
		procGetDiskFreeSpaceExW.Addr(),
		4,
		uintptr(unsafe.Pointer(directoryName)),
		uintptr(unsafe.Pointer(freeBytesAvailable)),
		uintptr(unsafe.Pointer(totalNumberOfBytes)),
		uintptr(unsafe.Pointer(totalNumberOfFreeBytes)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetDriveType(rootPathName *uint16) uint32 {
	r1, _, _ := syscall.Syscall(procGetDriveTypeW.Addr(), 1, uintptr(unsafe.Pointer(rootPathName)), 0, 0)
	return uint32(r1)
}

func GetEnvironmentStrings() (*uint16, error) {
	r1, _, e1 := syscall.Syscall(procGetEnvironmentStringsW.Addr(), 0, 0, 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return nil, e1
		} else {
			return nil, syscall.EINVAL
		}
	}
	return (*uint16)(unsafe.Pointer(r1)), nil
}

func GetEnvironmentVariable(name *uint16, buffer *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetEnvironmentVariableW.Addr(),
		3,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetFileAttributes(fileName *uint16) (uint32, error) {
	r1, _, e1 := syscall.Syscall(procGetFileAttributesW.Addr(), 1, uintptr(unsafe.Pointer(fileName)), 0, 0)
	if r1 == INVALID_FILE_ATTRIBUTES {
		if e1 != ERROR_SUCCESS {
			return uint32(r1), e1
		} else {
			return uint32(r1), syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetFileSize(file syscall.Handle, fileSizeHigh *uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetFileSize.Addr(),
		2,
		uintptr(file),
		uintptr(unsafe.Pointer(fileSizeHigh)),
		0)
	if r1 == INVALID_FILE_SIZE {
		if e1 != ERROR_SUCCESS {
			return uint32(r1), e1
		} else {
			return uint32(r1), syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetModuleFileName(module syscall.Handle, filename *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetModuleFileNameW.Addr(),
		3,
		uintptr(module),
		uintptr(unsafe.Pointer(filename)),
		uintptr(size))
	if r1 == 0 || r1 == uintptr(size) {
		if e1 != ERROR_SUCCESS {
			return uint32(r1), e1
		} else if r1 == uintptr(size) {
			return uint32(r1), ERROR_INSUFFICIENT_BUFFER
		} else {
			return uint32(r1), syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetProcessTimes(hProcess syscall.Handle, creationTime, exitTime, kernelTime, userTime *FILETIME) error {
	r1, _, e1 := syscall.Syscall6(
		procGetProcessTimes.Addr(),
		5,
		uintptr(hProcess),
		uintptr(unsafe.Pointer(creationTime)),
		uintptr(unsafe.Pointer(exitTime)),
		uintptr(unsafe.Pointer(kernelTime)),
		uintptr(unsafe.Pointer(userTime)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetStdHandle(stdHandle uint32) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(procGetStdHandle.Addr(), 1, uintptr(stdHandle), 0, 0)
	handle := (syscall.Handle)(r1)
	if handle == INVALID_HANDLE_VALUE {
		if e1 != ERROR_SUCCESS {
			return handle, e1
		} else {
			return handle, syscall.EINVAL
		}
	}
	return handle, nil
}

func GetSystemDirectory(buffer *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetSystemDirectoryW.Addr(),
		2,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetSystemInfo(systemInfo *SYSTEM_INFO) {
	syscall.Syscall(procGetSystemInfo.Addr(), 1, uintptr(unsafe.Pointer(systemInfo)), 0, 0)
}

func GetSystemTimeAsFileTime(systemTimeAsFileTime *FILETIME) {
	syscall.Syscall(procGetSystemTimeAsFileTime.Addr(), 1, uintptr(unsafe.Pointer(systemTimeAsFileTime)), 0, 0)
}

func GetSystemTimes(idleTime, kernelTime, userTime *FILETIME) error {
	r1, _, e1 := syscall.Syscall(
		procGetSystemTimes.Addr(),
		3,
		uintptr(unsafe.Pointer(idleTime)),
		uintptr(unsafe.Pointer(kernelTime)),
		uintptr(unsafe.Pointer(userTime)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetSystemWindowsDirectory(buffer *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetSystemWindowsDirectoryW.Addr(),
		2,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetSystemWow64Directory(buffer *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetSystemWow64DirectoryW.Addr(),
		2,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetTempFileName(pathName *uint16, prefixString *uint16, unique uint32, tempFileName *uint16) (uint32, error) {
	r1, _, e1 := syscall.Syscall6(
		procGetTempFileNameW.Addr(),
		4,
		uintptr(unsafe.Pointer(pathName)),
		uintptr(unsafe.Pointer(prefixString)),
		uintptr(unique),
		uintptr(unsafe.Pointer(tempFileName)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetTempPath(bufferLength uint32, buffer *uint16) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetTempPathW.Addr(),
		2,
		uintptr(bufferLength),
		uintptr(unsafe.Pointer(buffer)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func GetVersionEx(osvi *OSVERSIONINFOEX) error {
	r1, _, e1 := syscall.Syscall(procGetVersionExW.Addr(), 1, uintptr(unsafe.Pointer(osvi)), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetVolumeInformation(rootPathName *uint16, volumeNameBuffer *uint16, volumeNameSize uint32, volumeSerialNumber *uint32, maximumComponentLength *uint32, fileSystemFlags *uint32, fileSystemNameBuffer *uint16, fileSystemNameSize uint32) error {
	r1, _, e1 := syscall.Syscall9(
		procGetVolumeInformationW.Addr(),
		8,
		uintptr(unsafe.Pointer(rootPathName)),
		uintptr(unsafe.Pointer(volumeNameBuffer)),
		uintptr(volumeNameSize),
		uintptr(unsafe.Pointer(volumeSerialNumber)),
		uintptr(unsafe.Pointer(maximumComponentLength)),
		uintptr(unsafe.Pointer(fileSystemFlags)),
		uintptr(unsafe.Pointer(fileSystemNameBuffer)),
		uintptr(fileSystemNameSize),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetVolumeNameForVolumeMountPoint(volumeMountPoint *uint16, volumeName *uint16, bufferLength uint32) error {
	r1, _, e1 := syscall.Syscall(
		procGetVolumeNameForVolumeMountPointW.Addr(),
		3,
		uintptr(unsafe.Pointer(volumeMountPoint)),
		uintptr(unsafe.Pointer(volumeName)),
		uintptr(bufferLength))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetVolumePathName(fileName *uint16, volumePathName *uint16, bufferLength uint32) error {
	r1, _, e1 := syscall.Syscall(
		procGetVolumePathNameW.Addr(),
		3,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(unsafe.Pointer(volumePathName)),
		uintptr(bufferLength))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetWindowsDirectory(buffer *uint16, size uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procGetWindowsDirectoryW.Addr(),
		2,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func InitializeCriticalSection(criticalSection *CRITICAL_SECTION) {
	syscall.Syscall(procInitializeCriticalSection.Addr(), 1, uintptr(unsafe.Pointer(criticalSection)), 0, 0)
}

func IsProcessInJob(processHandle syscall.Handle, jobHandle syscall.Handle, result *bool) error {
	var resultRaw int32
	r1, _, e1 := syscall.Syscall(
		procIsProcessInJob.Addr(),
		3,
		uintptr(processHandle),
		uintptr(jobHandle),
		uintptr(unsafe.Pointer(&resultRaw)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	if result != nil {
		*result = (resultRaw != 0)
	}
	return nil
}

func LeaveCriticalSection(criticalSection *CRITICAL_SECTION) {
	syscall.Syscall(procLeaveCriticalSection.Addr(), 1, uintptr(unsafe.Pointer(criticalSection)), 0, 0)
}

func LoadLibrary(fileName *uint16) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(procLoadLibraryW.Addr(), 1, uintptr(unsafe.Pointer(fileName)), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func LocalFree(mem syscall.Handle) (syscall.Handle, error) {
	// LocalFree returns NULL to indicate success!
	r1, _, e1 := syscall.Syscall(procLocalFree.Addr(), 1, uintptr(mem), 0, 0)
	if r1 != 0 {
		if e1 != 0 {
			return syscall.Handle(r1), e1
		} else {
			return syscall.Handle(r1), syscall.EINVAL
		}
	}
	return 0, nil
}

func MoveFile(existingFileName *uint16, newFileName *uint16) error {
	r1, _, e1 := syscall.Syscall(
		procMoveFileW.Addr(),
		2,
		uintptr(unsafe.Pointer(existingFileName)),
		uintptr(unsafe.Pointer(newFileName)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func MoveFileEx(existingFileName *uint16, newFileName *uint16, flags uint32) error {
	r1, _, e1 := syscall.Syscall(
		procMoveFileExW.Addr(),
		3,
		uintptr(unsafe.Pointer(existingFileName)),
		uintptr(unsafe.Pointer(newFileName)),
		uintptr(flags))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func OpenJobObject(desiredAccess uint32, inheritHandle bool, name *uint16) (syscall.Handle, error) {
	var inheritHandleRaw int32
	if inheritHandle {
		inheritHandleRaw = 1
	} else {
		inheritHandleRaw = 0
	}
	r1, _, e1 := syscall.Syscall(
		procOpenJobObjectW.Addr(),
		3,
		uintptr(desiredAccess),
		uintptr(inheritHandleRaw),
		uintptr(unsafe.Pointer(name)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func OpenProcess(desiredAccess uint32, inheritHandle bool, processId uint32) (syscall.Handle, error) {
	var inheritHandleRaw int32
	if inheritHandle {
		inheritHandleRaw = 1
	} else {
		inheritHandleRaw = 0
	}
	r1, _, e1 := syscall.Syscall(
		procOpenProcess.Addr(),
		3,
		uintptr(desiredAccess),
		uintptr(inheritHandleRaw),
		uintptr(processId))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func QueryFullProcessImageName(process syscall.Handle, flags uint32, exeName *uint16, size *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procQueryFullProcessImageNameW.Addr(),
		4,
		uintptr(process),
		uintptr(flags),
		uintptr(unsafe.Pointer(exeName)),
		uintptr(unsafe.Pointer(size)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func QueryInformationJobObject(job syscall.Handle, jobObjectInfoClass int32, jobObjectInfo *byte, jobObjectInfoLength uint32, returnLength *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procQueryInformationJobObject.Addr(),
		5,
		uintptr(job),
		uintptr(jobObjectInfoClass),
		uintptr(unsafe.Pointer(jobObjectInfo)),
		uintptr(jobObjectInfoLength),
		uintptr(unsafe.Pointer(returnLength)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func ReadFile(file syscall.Handle, buffer *byte, numberOfBytesToRead uint32, numberOfBytesRead *uint32, overlapped *OVERLAPPED) error {
	r1, _, e1 := syscall.Syscall6(
		procReadFile.Addr(),
		5,
		uintptr(file),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(numberOfBytesToRead),
		uintptr(unsafe.Pointer(numberOfBytesRead)),
		uintptr(unsafe.Pointer(overlapped)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func ReadProcessMemory(process syscall.Handle, baseAddress uintptr, buffer *byte, size uint32, numberOfBytesRead *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procReadProcessMemory.Addr(),
		5,
		uintptr(process),
		baseAddress,
		uintptr(unsafe.Pointer(buffer)),
		uintptr(size),
		uintptr(unsafe.Pointer(numberOfBytesRead)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func SetEnvironmentVariable(name *uint16, value *uint16) error {
	r1, _, e1 := syscall.Syscall(
		procSetEnvironmentVariableW.Addr(),
		2,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(value)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func SetFileAttributes(fileName *uint16, fileAttributes uint32) error {
	r1, _, e1 := syscall.Syscall(
		procSetFileAttributesW.Addr(),
		2,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(fileAttributes),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func SetFileTime(file syscall.Handle, creationTime *FILETIME, lastAccessTime *FILETIME, lastWriteTime *FILETIME) error {
	r1, _, e1 := syscall.Syscall6(
		procSetFileTime.Addr(),
		4,
		uintptr(file),
		uintptr(unsafe.Pointer(creationTime)),
		uintptr(unsafe.Pointer(lastAccessTime)),
		uintptr(unsafe.Pointer(lastWriteTime)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func SetInformationJobObject(job syscall.Handle, jobObjectInfoClass int32, jobObjectInfo *byte, jobObjectInfoLength uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procSetInformationJobObject.Addr(),
		4,
		uintptr(job),
		uintptr(jobObjectInfoClass),
		uintptr(unsafe.Pointer(jobObjectInfo)),
		uintptr(jobObjectInfoLength),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func SetStdHandle(stdHandle uint32, handle syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(
		procSetStdHandle.Addr(),
		2,
		uintptr(stdHandle),
		uintptr(handle),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func TerminateJobObject(job syscall.Handle, exitCode uint32) error {
	r1, _, e1 := syscall.Syscall(
		procTerminateJobObject.Addr(),
		2,
		uintptr(job),
		uintptr(exitCode),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func TerminateProcess(process syscall.Handle, exitCode uint32) error {
	r1, _, e1 := syscall.Syscall(
		procTerminateProcess.Addr(),
		2,
		uintptr(process),
		uintptr(exitCode),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func TryEnterCriticalSection(criticalSection *CRITICAL_SECTION) bool {
	r1, _, _ := syscall.Syscall(
		procTryEnterCriticalSection.Addr(),
		1,
		uintptr(unsafe.Pointer(criticalSection)),
		0,
		0)
	return r1 != 0
}

func UpdateResource(update syscall.Handle, resourceType uintptr, name uintptr, language uint16, data *byte, cbData uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procUpdateResourceW.Addr(),
		6,
		uintptr(update),
		resourceType,
		name,
		uintptr(language),
		uintptr(unsafe.Pointer(data)),
		uintptr(cbData))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func VerifyVersionInfo(versionInfo *OSVERSIONINFOEX, typeMask uint32, conditionMask uint64) error {
	r1, _, e1 := syscall.Syscall(
		procVerifyVersionInfoW.Addr(),
		3,
		uintptr(unsafe.Pointer(versionInfo)),
		uintptr(typeMask),
		uintptr(conditionMask))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func WaitForSingleObject(handle syscall.Handle, milliseconds uint32) (uint32, error) {
	r1, _, e1 := syscall.Syscall(
		procWaitForSingleObject.Addr(),
		2,
		uintptr(handle),
		uintptr(milliseconds),
		0)
	if r1 == WAIT_FAILED {
		if e1 != ERROR_SUCCESS {
			return uint32(r1), e1
		} else {
			return uint32(r1), syscall.EINVAL
		}
	}
	return uint32(r1), nil
}

func Lstrlen(string *uint16) int32 {
	r1, _, _ := syscall.Syscall(proclstrlenW.Addr(), 1, uintptr(unsafe.Pointer(string)), 0, 0)
	return int32(r1)
}

func AdjustTokenPrivileges(tokenHandle syscall.Handle, disableAllPrivileges bool, newState *TOKEN_PRIVILEGES, bufferLength uint32, previousState *TOKEN_PRIVILEGES, returnLength *uint32) error {
	var disableAllPrivilegesRaw int32
	if disableAllPrivileges {
		disableAllPrivilegesRaw = 1
	} else {
		disableAllPrivilegesRaw = 0
	}
	r1, _, e1 := syscall.Syscall6(
		procAdjustTokenPrivileges.Addr(),
		6,
		uintptr(tokenHandle),
		uintptr(disableAllPrivilegesRaw),
		uintptr(unsafe.Pointer(newState)),
		uintptr(bufferLength),
		uintptr(unsafe.Pointer(previousState)),
		uintptr(unsafe.Pointer(returnLength)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func AllocateAndInitializeSid(identifierAuthority *SID_IDENTIFIER_AUTHORITY, subAuthorityCount byte, subAuthority0 uint32, subAuthority1 uint32, subAuthority2 uint32, subAuthority3 uint32, subAuthority4 uint32, subAuthority5 uint32, subAuthority6 uint32, subAuthority7 uint32, sid **SID) error {
	r1, _, e1 := syscall.Syscall12(
		procAllocateAndInitializeSid.Addr(),
		11,
		uintptr(unsafe.Pointer(identifierAuthority)),
		uintptr(subAuthorityCount),
		uintptr(subAuthority0),
		uintptr(subAuthority1),
		uintptr(subAuthority2),
		uintptr(subAuthority3),
		uintptr(subAuthority4),
		uintptr(subAuthority5),
		uintptr(subAuthority6),
		uintptr(subAuthority7),
		uintptr(unsafe.Pointer(sid)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CheckTokenMembership(tokenHandle syscall.Handle, sidToCheck *SID, isMember *bool) error {
	var isMemberRaw int32
	r1, _, e1 := syscall.Syscall(
		procCheckTokenMembership.Addr(),
		3,
		uintptr(tokenHandle),
		uintptr(unsafe.Pointer(sidToCheck)),
		uintptr(unsafe.Pointer(&isMemberRaw)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	if isMember != nil {
		*isMember = (isMemberRaw != 0)
	}
	return nil
}

func CopySid(destinationSidLength uint32, destinationSid *SID, sourceSid *SID) error {
	r1, _, e1 := syscall.Syscall(
		procCopySid.Addr(),
		3,
		uintptr(destinationSidLength),
		uintptr(unsafe.Pointer(destinationSid)),
		uintptr(unsafe.Pointer(sourceSid)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func DeregisterEventSource(eventLog syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procDeregisterEventSource.Addr(), 1, uintptr(eventLog), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func EqualSid(sid1 *SID, sid2 *SID) bool {
	r1, _, _ := syscall.Syscall(
		procEqualSid.Addr(),
		2,
		uintptr(unsafe.Pointer(sid1)),
		uintptr(unsafe.Pointer(sid2)),
		0)
	return r1 != 0
}

func FreeSid(sid *SID) {
	syscall.Syscall(procFreeSid.Addr(), 1, uintptr(unsafe.Pointer(sid)), 0, 0)
}

func GetFileSecurity(fileName *uint16, requestedInformation uint32, securityDescriptor *byte, length uint32, lengthNeeded *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procGetFileSecurityW.Addr(),
		5,
		uintptr(unsafe.Pointer(fileName)),
		uintptr(requestedInformation),
		uintptr(unsafe.Pointer(securityDescriptor)),
		uintptr(length),
		uintptr(unsafe.Pointer(lengthNeeded)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func GetLengthSid(sid *SID) uint32 {
	r1, _, _ := syscall.Syscall(procGetLengthSid.Addr(), 1, uintptr(unsafe.Pointer(sid)), 0, 0)
	return uint32(r1)
}

func GetSecurityDescriptorOwner(securityDescriptor *byte, owner **SID, ownerDefaulted *bool) error {
	var ownerDefaultedRaw int32
	r1, _, e1 := syscall.Syscall(
		procGetSecurityDescriptorOwner.Addr(),
		3,
		uintptr(unsafe.Pointer(securityDescriptor)),
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(&ownerDefaultedRaw)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	if ownerDefaulted != nil {
		*ownerDefaulted = (ownerDefaultedRaw != 0)
	}
	return nil
}

func GetTokenInformation(tokenHandle syscall.Handle, tokenInformationClass int32, tokenInformation *byte, tokenInformationLength uint32, returnLength *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procGetTokenInformation.Addr(),
		5,
		uintptr(tokenHandle),
		uintptr(tokenInformationClass),
		uintptr(unsafe.Pointer(tokenInformation)),
		uintptr(tokenInformationLength),
		uintptr(unsafe.Pointer(returnLength)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func ImpersonateSelf(impersonationLevel int32) error {
	r1, _, e1 := syscall.Syscall(procImpersonateSelf.Addr(), 1, uintptr(impersonationLevel), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func LookupAccountName(systemName *uint16, accountName *uint16, sid *SID, cbSid *uint32, referencedDomainName *uint16, cchReferencedDomainName *uint32, use *int32) error {
	r1, _, e1 := syscall.Syscall9(
		procLookupAccountNameW.Addr(),
		7,
		uintptr(unsafe.Pointer(systemName)),
		uintptr(unsafe.Pointer(accountName)),
		uintptr(unsafe.Pointer(sid)),
		uintptr(unsafe.Pointer(cbSid)),
		uintptr(unsafe.Pointer(referencedDomainName)),
		uintptr(unsafe.Pointer(cchReferencedDomainName)),
		uintptr(unsafe.Pointer(use)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func LookupPrivilegeValue(systemName *uint16, name *uint16, luid *LUID) error {
	r1, _, e1 := syscall.Syscall(
		procLookupPrivilegeValueW.Addr(),
		3,
		uintptr(unsafe.Pointer(systemName)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(luid)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func OpenProcessToken(processHandle syscall.Handle, desiredAccess uint32, tokenHandle *syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(
		procOpenProcessToken.Addr(),
		3,
		uintptr(processHandle),
		uintptr(desiredAccess),
		uintptr(unsafe.Pointer(tokenHandle)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func OpenThreadToken(threadHandle syscall.Handle, desiredAccess uint32, openAsSelf bool, tokenHandle *syscall.Handle) error {
	var openAsSelfRaw int32
	if openAsSelf {
		openAsSelfRaw = 1
	} else {
		openAsSelfRaw = 0
	}
	r1, _, e1 := syscall.Syscall6(
		procOpenThreadToken.Addr(),
		4,
		uintptr(threadHandle),
		uintptr(desiredAccess),
		uintptr(openAsSelfRaw),
		uintptr(unsafe.Pointer(tokenHandle)),
		0,
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func RegisterEventSource(uncServerName *uint16, sourceName *uint16) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procRegisterEventSourceW.Addr(),
		2,
		uintptr(unsafe.Pointer(uncServerName)),
		uintptr(unsafe.Pointer(sourceName)),
		0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func ReportEvent(eventLog syscall.Handle, eventType uint16, category uint16, eventID uint32, userSid *SID, numStrings uint16, dataSize uint32, strings **uint16, rawData *byte) error {
	r1, _, e1 := syscall.Syscall9(
		procReportEventW.Addr(),
		9,
		uintptr(eventLog),
		uintptr(eventType),
		uintptr(category),
		uintptr(eventID),
		uintptr(unsafe.Pointer(userSid)),
		uintptr(numStrings),
		uintptr(dataSize),
		uintptr(unsafe.Pointer(strings)),
		uintptr(unsafe.Pointer(rawData)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func RevertToSelf() error {
	r1, _, e1 := syscall.Syscall(procRevertToSelf.Addr(), 0, 0, 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}
