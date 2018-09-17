package syscallex

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// JobObjectInfoClass
// cf. https://msdn.microsoft.com/en-us/library/windows/desktop/ms686216%28v=vs.85%29.aspx?f=255&MSPPError=-2147217396
const (
	JobObjectInfoClass_JobObjectBasicProcessIdList                 = 3
	JobObjectInfoClass_JobObjectAssociateCompletionPortInformation = 7
	JobObjectInfoClass_JobObjectExtendedLimitInformation           = 9
)

// JobObjectBasicLimitInformation.LimitFlags
const (
	JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE = 0x00002000
)

// job object completion statuses, thanks wine!
// cf. https://www.winehq.org/pipermail/wine-cvs/2013-October/097834.html
const (
	JOB_OBJECT_MSG_END_OF_JOB_TIME       = 1
	JOB_OBJECT_MSG_END_OF_PROCESS_TIME   = 2
	JOB_OBJECT_MSG_ACTIVE_PROCESS_LIMIT  = 3
	JOB_OBJECT_MSG_ACTIVE_PROCESS_ZERO   = 4
	JOB_OBJECT_MSG_NEW_PROCESS           = 6
	JOB_OBJECT_MSG_EXIT_PROCESS          = 7
	JOB_OBJECT_MSG_ABNORMAL_EXIT_PROCESS = 8
	JOB_OBJECT_MSG_PROCESS_MEMORY_LIMIT  = 9
	JOB_OBJECT_MSG_JOB_MEMORY_LIMIT      = 10
)

type JobObjectAssociateCompletionPort struct {
	CompletionKey  syscall.Handle
	CompletionPort syscall.Handle
}

const (
	CREATE_SUSPENDED      = 0x00000004
	CREATE_NEW_CONSOLE    = 0x00000010
	PROCESS_ALL_ACCESS    = syscall.STANDARD_RIGHTS_REQUIRED | syscall.SYNCHRONIZE | 0xfff
	THREAD_SUSPEND_RESUME = 0x0002

	TH32CS_SNAPPROCESS = 0x00000002
)

type ThreadEntry32 struct {
	Size           uint32
	TUsage         uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

type ProcessEntry32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

var (
	modkernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procCreateJobObject           = modkernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject   = modkernel32.NewProc("SetInformationJobObject")
	procQueryInformationJobObject = modkernel32.NewProc("QueryInformationJobObject")
	procAssignProcessToJobObject  = modkernel32.NewProc("AssignProcessToJobObject")

	procGetCurrentThread    = modkernel32.NewProc("GetCurrentThread")
	procOpenThreadToken     = modkernel32.NewProc("OpenThreadToken")
	procGetDiskFreeSpaceExW = modkernel32.NewProc("GetDiskFreeSpaceExW")

	procOpenThread    = modkernel32.NewProc("OpenThread")
	procResumeThread  = modkernel32.NewProc("ResumeThread")
	procThread32First = modkernel32.NewProc("Thread32First")
	procThread32Next  = modkernel32.NewProc("Thread32Next")

	procCreateToolhelp32Snapshot = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modkernel32.NewProc("Process32FirstW")
	procProcess32NextW           = modkernel32.NewProc("Process32NextW")

	procQueryFullProcessImageNameW = modkernel32.NewProc("QueryFullProcessImageNameW")
)

func CreateJobObject(
	jobAttributes *syscall.SecurityAttributes,
	name *uint16,
) (handle syscall.Handle, err error) {
	r1, _, e1 := syscall.Syscall(
		procCreateJobObject.Addr(),
		2,
		uintptr(unsafe.Pointer(jobAttributes)),
		uintptr(unsafe.Pointer(name)),
		0,
	)
	handle = syscall.Handle(r1)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

type IoCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

func SetInformationJobObject(
	jobObject syscall.Handle,
	jobObjectInfoClass uint32,
	jobObjectInfo uintptr,
	jobObjectInfoLength uintptr,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procSetInformationJobObject.Addr(),
		4,
		uintptr(jobObject),
		uintptr(jobObjectInfoClass),
		jobObjectInfo,
		jobObjectInfoLength,
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func QueryInformationJobObject(
	jobObject syscall.Handle,
	jobObjectInfoClass uint32,
	jobObjectInfo uintptr,
	jobObjectInfoLength uintptr,
	returnLength uintptr,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procQueryInformationJobObject.Addr(),
		5,
		uintptr(jobObject),
		uintptr(jobObjectInfoClass),
		jobObjectInfo,
		jobObjectInfoLength,
		returnLength,
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func AssignProcessToJobObject(
	jobObject syscall.Handle,
	process syscall.Handle,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procAssignProcessToJobObject.Addr(),
		2,
		uintptr(jobObject),
		uintptr(process),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func GetCurrentThread() syscall.Handle {
	r1, _, _ := syscall.Syscall(
		procGetCurrentThread.Addr(),
		0,
		0, 0, 0,
	)
	return syscall.Handle(r1)
}

func OpenThreadToken(
	threadHandle syscall.Handle,
	desiredAccess uint32,
	openAsSelf uint32,
	tokenHandle *syscall.Token,
) (err error) {
	r1, _, e1 := syscall.Syscall6(
		procOpenThreadToken.Addr(),
		4,
		uintptr(threadHandle),
		uintptr(desiredAccess),
		uintptr(openAsSelf),
		uintptr(unsafe.Pointer(tokenHandle)),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

type DiskFreeSpace struct {
	FreeBytesAvailable     uint64
	TotalNumberOfBytes     uint64
	TotalNumberOfFreeBytes uint64
}

func GetDiskFreeSpaceEx(path *uint16) (dfs *DiskFreeSpace, err error) {
	var buf DiskFreeSpace
	dfs = &buf

	r1, _, e1 := syscall.Syscall6(
		procGetDiskFreeSpaceExW.Addr(),
		4,
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&buf.FreeBytesAvailable)),
		uintptr(unsafe.Pointer(&buf.TotalNumberOfBytes)),
		uintptr(unsafe.Pointer(&buf.TotalNumberOfFreeBytes)),
		0, 0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return dfs, err
}

func OpenThread(
	desiredAccess uint32,
	inheritHandle uint32,
	threadId uint32,
) (handle syscall.Handle, err error) {
	r1, _, e1 := syscall.Syscall(
		procOpenThread.Addr(),
		3,
		uintptr(desiredAccess),
		uintptr(inheritHandle),
		uintptr(threadId),
	)
	handle = syscall.Handle(r1)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func ResumeThread(
	thread syscall.Handle,
) (retCount uint32, err error) {
	r1, _, e1 := syscall.Syscall(
		procResumeThread.Addr(),
		1,
		uintptr(thread),
		0,
		0,
	)

	minusOne := int(-1)
	if r1 == uintptr(minusOne) {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	} else {
		retCount = uint32(r1)
	}
	return
}

func Thread32First(
	snapshot syscall.Handle,
	pThreadEntry *ThreadEntry32,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procThread32First.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pThreadEntry)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func Thread32Next(
	snapshot syscall.Handle,
	pThreadEntry *ThreadEntry32,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procThread32Next.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pThreadEntry)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func CreateToolhelp32Snapshot(
	flags uint32,
	processID uint32,
) (handle syscall.Handle, err error) {
	r1, _, e1 := syscall.Syscall(
		procCreateToolhelp32Snapshot.Addr(),
		2,
		uintptr(flags),
		uintptr(processID),
		0,
	)
	handle = syscall.Handle(r1)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func Process32First(
	snapshot syscall.Handle,
	pProcessEntry *ProcessEntry32,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procProcess32FirstW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pProcessEntry)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func Process32Next(
	snapshot syscall.Handle,
	pProcessEntry *ProcessEntry32,
) (err error) {
	r1, _, e1 := syscall.Syscall(
		procProcess32NextW.Addr(),
		2,
		uintptr(snapshot),
		uintptr(unsafe.Pointer(pProcessEntry)),
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func QueryFullProcessImageName(
	process syscall.Handle,
	flags uint32,
) (s string, err error) {
	var bufferSize uint32 = 32 * 1024
	buffer := make([]uint16, bufferSize)

	r1, _, e1 := syscall.Syscall6(
		procQueryFullProcessImageNameW.Addr(),
		4,
		uintptr(process),
		uintptr(flags),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferSize)),
		0,
		0,
	)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}
	if err == nil {
		s = syscall.UTF16ToString(buffer[:bufferSize])
	}
	return
}
