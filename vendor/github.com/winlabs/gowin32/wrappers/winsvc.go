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
	SERVICES_ACTIVE_DATABASE = "ServicesActive"
)

const (
	SERVICE_NO_CHANGE = 0xFFFFFFFF
)

const (
	SERVICE_ACTIVE    = 0x00000001
	SERVICE_INACTIVE  = 0x00000002
	SERVICE_STATE_ALL = SERVICE_ACTIVE | SERVICE_INACTIVE
)

const (
	SERVICE_CONTROL_STOP                  = 0x00000001
	SERVICE_CONTROL_PAUSE                 = 0x00000002
	SERVICE_CONTROL_CONTINUE              = 0x00000003
	SERVICE_CONTROL_INTERROGATE           = 0x00000004
	SERVICE_CONTROL_SHUTDOWN              = 0x00000005
	SERVICE_CONTROL_PARAMCHANGE           = 0x00000006
	SERVICE_CONTROL_NETBINDADD            = 0x00000007
	SERVICE_CONTROL_NETBINDREMOVE         = 0x00000008
	SERVICE_CONTROL_NETBINDENABLE         = 0x00000009
	SERVICE_CONTROL_NETBINDDISABLE        = 0x0000000A
	SERVICE_CONTROL_DEVICEEVENT           = 0x0000000B
	SERVICE_CONTROL_HARDWAREPROFILECHANGE = 0x0000000C
	SERVICE_CONTROL_POWEREVENT            = 0x0000000D
	SERVICE_CONTROL_SESSIONCHANGE         = 0x0000000E
	SERVICE_CONTROL_PRESHUTDOWN           = 0x0000000F
	SERVICE_CONTROL_TIMECHANGE            = 0x00000010
	SERVICE_CONTROL_TRIGGEREVENT          = 0x00000020
)

const (
	SERVICE_STOPPED          = 0x00000001
	SERVICE_START_PENDING    = 0x00000002
	SERVICE_STOP_PENDING     = 0x00000003
	SERVICE_RUNNING          = 0x00000004
	SERVICE_CONTINUE_PENDING = 0x00000005
	SERVICE_PAUSE_PENDING    = 0x00000006
	SERVICE_PAUSED           = 0x00000007
)

const (
	SERVICE_ACCEPT_STOP                  = 0x00000001
	SERVICE_ACCEPT_PAUSE_CONTINUE        = 0x00000002
	SERVICE_ACCEPT_SHUTDOWN              = 0x00000004
	SERVICE_ACCEPT_PARAMCHANGE           = 0x00000008
	SERVICE_ACCEPT_NETBINDCHANGE         = 0x00000010
	SERVICE_ACCEPT_HARDWAREPROFILECHANGE = 0x00000020
	SERVICE_ACCEPT_POWEREVENT            = 0x00000040
	SERVICE_ACCEPT_SESSIONCHANGE         = 0x00000080
	SERVICE_ACCEPT_PRESHUTDOWN           = 0x00000100
	SERVICE_ACCEPT_TIMECHANGE            = 0x00000200
	SERVICE_ACCEPT_TRIGGEREVENT          = 0x00000400
)

const (
	SC_MANAGER_CONNECT            = 0x0001
	SC_MANAGER_CREATE_SERVICE     = 0x0002
	SC_MANAGER_ENUMERATE_SERVICE  = 0x0004
	SC_MANAGER_LOCK               = 0x0008
	SC_MANAGER_QUERY_LOCK_STATUS  = 0x0010
	SC_MANAGER_MODIFY_BOOT_CONFIG = 0x0020
	SC_MANAGER_ALL_ACCESS         = STANDARD_RIGHTS_REQUIRED | SC_MANAGER_CONNECT | SC_MANAGER_CREATE_SERVICE | SC_MANAGER_ENUMERATE_SERVICE | SC_MANAGER_LOCK | SC_MANAGER_QUERY_LOCK_STATUS | SC_MANAGER_MODIFY_BOOT_CONFIG
)

const (
	SERVICE_QUERY_CONFIG         = 0x0001
	SERVICE_CHANGE_CONFIG        = 0x0002
	SERVICE_QUERY_STATUS         = 0x0004
	SERVICE_ENUMERATE_DEPENDENTS = 0x0008
	SERVICE_START                = 0x0010
	SERVICE_STOP                 = 0x0020
	SERVICE_PAUSE_CONTINUE       = 0x0040
	SERVICE_INTERROGATE          = 0x0080
	SERVICE_USER_DEFINED_CONTROL = 0x0100
	SERVICE_ALL_ACCESS           = STANDARD_RIGHTS_REQUIRED | SERVICE_QUERY_CONFIG | SERVICE_CHANGE_CONFIG | SERVICE_QUERY_STATUS | SERVICE_ENUMERATE_DEPENDENTS | SERVICE_START | SERVICE_STOP | SERVICE_PAUSE_CONTINUE | SERVICE_INTERROGATE | SERVICE_USER_DEFINED_CONTROL
)

const (
	SERVICE_RUNS_IN_SYSTEM_PROCESS = 0x00000001
)

const (
	SERVICE_CONFIG_DESCRIPTION              = 1
	SERVICE_CONFIG_FAILURE_ACTIONS          = 2
	SERVICE_CONFIG_DELAYED_AUTO_START_INFO  = 3
	SERVICE_CONFIG_FAILURE_ACTIONS_FLAG     = 4
	SERVICE_CONFIG_SERVICE_SID_INFO         = 5
	SERVICE_CONFIG_REQUIRED_PRIVILEGES_INFO = 6
	SERVICE_CONFIG_PRESHUTDOWN_INFO         = 7
	SERVICE_CONFIG_TRIGGER_INFO             = 8
	SERVICE_CONFIG_PREFERRED_NODE           = 9
)

const (
	SERVICE_SID_TYPE_NONE         = 0x00000000
	SERVICE_SID_TYPE_UNRESTRICTED = 0x00000001
	SERVICE_SID_TYPE_RESTRICTED   = 0x00000002 | SERVICE_SID_TYPE_UNRESTRICTED
)

type SERVICE_DESCRIPTION struct {
	Description *uint16
}

const (
	SC_ACTION_NONE        = 0
	SC_ACTION_RESTART     = 1
	SC_ACTION_REBOOT      = 2
	SC_ACTION_RUN_COMMAND = 3
)

type SC_ACTION struct {
	Type  int32
	Delay uint32
}

type SERVICE_FAILURE_ACTIONS struct {
	ResetPeriod uint32
	RebootMsg   *uint16
	Command     *uint16
	CActions    uint32
	Actions     *SC_ACTION
}

type SERVICE_DELAYED_AUTO_START_INFO struct {
	DelayedAutostart int32
}

type SERVICE_FAILURE_ACTIONS_FLAG struct {
	FailureActionsOnNonCrashFailures int32
}

type SERVICE_SID_INFO struct {
	ServiceSidType uint32
}

type SERVICE_REQUIRED_PRIVILEGES_INFO struct {
	RequiredPrivileges *uint16
}

type SERVICE_PRESHUTDOWN_INFO struct {
	PreshutdownTimeout uint32
}

type SERVICE_TRIGGER_SPECIFIC_DATA_ITEM struct {
	DataType uint32
	CbData   uint32
	Data     *byte
}

type SERVICE_TRIGGER struct {
	TriggerType    uint32
	Action         uint32
	TriggerSubtype *GUID
	CDataItems     uint32
	DataItems      *SERVICE_TRIGGER_SPECIFIC_DATA_ITEM
}

type SERVICE_TRIGGER_INFO struct {
	CTriggers uint32
	Triggers  *SERVICE_TRIGGER
	Reserved  *byte
}

type SERVICE_PREFERRED_NODE_INFO struct {
	PreferredNode uint16
	Delete        uint16
}

const (
	SC_STATUS_PROCESS_INFO = 0
)

type SERVICE_STATUS struct {
	ServiceType             uint32
	CurrentState            uint32
	ControlsAccepted        uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	CheckPoint              uint32
	WaitHint                uint32
}

type SERVICE_STATUS_PROCESS struct {
	ServiceType             uint32
	CurrentState            uint32
	ControlsAccepted        uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	CheckPoint              uint32
	WaitHint                uint32
	ProcessId               uint32
	ServiceFlags            uint32
}

type ENUM_SERVICE_STATUS struct {
	ServiceName   *uint16
	DisplayName   *uint16
	ServiceStatus SERVICE_STATUS
}

type QUERY_SERVICE_CONFIG struct {
	ServiceType      uint32
	StartType        uint32
	ErrorControl     uint32
	BinaryPathName   *uint16
	LoadOrderGroup   *uint16
	TagId            uint32
	Dependencies     *uint16
	ServiceStartName *uint16
	DisplayName      *uint16
}

var (
	procChangeServiceConfig2W = modadvapi32.NewProc("ChangeServiceConfig2W")
	procChangeServiceConfigW  = modadvapi32.NewProc("ChangeServiceConfigW")
	procCloseServiceHandle    = modadvapi32.NewProc("CloseServiceHandle")
	procControlService        = modadvapi32.NewProc("ControlService")
	procDeleteService         = modadvapi32.NewProc("DeleteService")
	procCreateServiceW        = modadvapi32.NewProc("CreateServiceW")
	procEnumServicesStatusW   = modadvapi32.NewProc("EnumServicesStatusW")
	procOpenSCManagerW        = modadvapi32.NewProc("OpenSCManagerW")
	procOpenServiceW          = modadvapi32.NewProc("OpenServiceW")
	procQueryServiceConfig2W  = modadvapi32.NewProc("QueryServiceConfig2W")
	procQueryServiceConfigW   = modadvapi32.NewProc("QueryServiceConfigW")
	procQueryServiceStatus    = modadvapi32.NewProc("QueryServiceStatus")
	procQueryServiceStatusEx  = modadvapi32.NewProc("QueryServiceStatusEx")
	procStartServiceW         = modadvapi32.NewProc("StartServiceW")
)

func ChangeServiceConfig(service syscall.Handle, serviceType uint32, startType uint32, errorControl uint32, binaryPathName *uint16, loadOrderGroup *uint16, tagId *uint32, dependencies *uint16, serviceStartName *uint16, password *uint16, displayName *uint16) error {
	r1, _, e1 := syscall.Syscall12(
		procChangeServiceConfigW.Addr(),
		11,
		uintptr(service),
		uintptr(serviceType),
		uintptr(startType),
		uintptr(errorControl),
		uintptr(unsafe.Pointer(binaryPathName)),
		uintptr(unsafe.Pointer(loadOrderGroup)),
		uintptr(unsafe.Pointer(tagId)),
		uintptr(unsafe.Pointer(dependencies)),
		uintptr(unsafe.Pointer(serviceStartName)),
		uintptr(unsafe.Pointer(password)),
		uintptr(unsafe.Pointer(displayName)),
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

func ChangeServiceConfig2(service syscall.Handle, infoLevel uint32, info *byte) error {
	r1, _, e1 := syscall.Syscall(
		procChangeServiceConfig2W.Addr(),
		3,
		uintptr(service),
		uintptr(infoLevel),
		uintptr(unsafe.Pointer(info)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CloseServiceHandle(scObject syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procCloseServiceHandle.Addr(), 1, uintptr(scObject), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func ControlService(service syscall.Handle, control uint32, serviceStatus *SERVICE_STATUS) error {
	r1, _, e1 := syscall.Syscall(
		procControlService.Addr(),
		3,
		uintptr(service),
		uintptr(control),
		uintptr(unsafe.Pointer(serviceStatus)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func CreateService(scManager syscall.Handle, serviceName *uint16, databaseName *uint16, desiredAccess uint32, serviceType uint32, startType uint32, errorControl uint32, binaryPathName *uint16, loadOrderGroup *uint16, tagId *uint32, dependencies *uint16, serviceStartName *uint16, password *uint16) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall15(
		procCreateServiceW.Addr(),
		13,
		uintptr(scManager),
		uintptr(unsafe.Pointer(serviceName)),
		uintptr(unsafe.Pointer(databaseName)),
		uintptr(desiredAccess),
		uintptr(serviceType),
		uintptr(startType),
		uintptr(errorControl),
		uintptr(unsafe.Pointer(binaryPathName)),
		uintptr(unsafe.Pointer(loadOrderGroup)),
		uintptr(unsafe.Pointer(tagId)),
		uintptr(unsafe.Pointer(dependencies)),
		uintptr(unsafe.Pointer(serviceStartName)),
		uintptr(unsafe.Pointer(password)),
		0,
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

func DeleteService(service syscall.Handle) error {
	r1, _, e1 := syscall.Syscall(procDeleteService.Addr(), 1, uintptr(service), 0, 0)
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func EnumServicesStatus(scManager syscall.Handle, serviceType uint32, serviceState uint32, services *byte, bufSize uint32, bytesNeeded *uint32, servicesReturned *uint32, resumeHandle *uint32) error {
	r1, _, e1 := syscall.Syscall9(
		procEnumServicesStatusW.Addr(),
		8,
		uintptr(scManager),
		uintptr(serviceType),
		uintptr(serviceState),
		uintptr(unsafe.Pointer(services)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(bytesNeeded)),
		uintptr(unsafe.Pointer(servicesReturned)),
		uintptr(unsafe.Pointer(resumeHandle)),
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

func OpenSCManager(machineName *uint16, databaseName *uint16, desiredAccess uint32) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procOpenSCManagerW.Addr(),
		3,
		uintptr(unsafe.Pointer(machineName)),
		uintptr(unsafe.Pointer(databaseName)),
		uintptr(desiredAccess))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func OpenService(scManager syscall.Handle, serviceName *uint16, desiredAccess uint32) (syscall.Handle, error) {
	r1, _, e1 := syscall.Syscall(
		procOpenServiceW.Addr(),
		3,
		uintptr(scManager),
		uintptr(unsafe.Pointer(serviceName)),
		uintptr(desiredAccess))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return 0, e1
		} else {
			return 0, syscall.EINVAL
		}
	}
	return syscall.Handle(r1), nil
}

func QueryServiceConfig(service syscall.Handle, serviceConfig *QUERY_SERVICE_CONFIG, bufSize uint32, bytesNeeded *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procQueryServiceConfigW.Addr(),
		4,
		uintptr(service),
		uintptr(unsafe.Pointer(serviceConfig)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(bytesNeeded)),
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

func QueryServiceConfig2(service syscall.Handle, infoLevel uint32, buffer *byte, bufSize uint32, bytesNeeded *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procQueryServiceConfig2W.Addr(),
		5,
		uintptr(service),
		uintptr(infoLevel),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(bytesNeeded)),
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

func QueryServiceStatus(service syscall.Handle, serviceStatus *SERVICE_STATUS) error {
	r1, _, e1 := syscall.Syscall(
		procQueryServiceStatus.Addr(),
		2,
		uintptr(service),
		uintptr(unsafe.Pointer(serviceStatus)),
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

func QueryServiceStatusEx(service syscall.Handle, infoLevel int32, buffer *byte, bufSize uint32, bytesNeeded *uint32) error {
	r1, _, e1 := syscall.Syscall6(
		procQueryServiceStatusEx.Addr(),
		5,
		uintptr(service),
		uintptr(infoLevel),
		uintptr(unsafe.Pointer(buffer)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(bytesNeeded)),
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

func StartService(service syscall.Handle, numServiceArgs uint32, serviceArgVectors **uint16) error {
	r1, _, e1 := syscall.Syscall(
		procStartServiceW.Addr(),
		3,
		uintptr(service),
		uintptr(numServiceArgs),
		uintptr(unsafe.Pointer(serviceArgVectors)))
	if r1 == 0 {
		if e1 != ERROR_SUCCESS {
			return e1
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}
