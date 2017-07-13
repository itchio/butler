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

	"syscall"
	"unsafe"
)

type ServiceType uint32

const (
	ServiceKernelDriver       ServiceType = wrappers.SERVICE_KERNEL_DRIVER
	ServiceFileSystemDriver   ServiceType = wrappers.SERVICE_FILE_SYSTEM_DRIVER
	ServiceAdapter            ServiceType = wrappers.SERVICE_ADAPTER
	ServiceRecognizerDriver   ServiceType = wrappers.SERVICE_RECOGNIZER_DRIVER
	ServiceDriver             ServiceType = wrappers.SERVICE_DRIVER
	ServiceWin32OwnProcess    ServiceType = wrappers.SERVICE_WIN32_OWN_PROCESS
	ServiceWin32ShareProcess  ServiceType = wrappers.SERVICE_WIN32_SHARE_PROCESS
	ServiceWin32              ServiceType = wrappers.SERVICE_WIN32
	ServiceInteractiveProcess ServiceType = wrappers.SERVICE_INTERACTIVE_PROCESS
)

type ServiceStartType uint32

const (
	ServiceBootStart   ServiceStartType = wrappers.SERVICE_BOOT_START
	ServiceSystemStart ServiceStartType = wrappers.SERVICE_SYSTEM_START
	ServiceAutoStart   ServiceStartType = wrappers.SERVICE_AUTO_START
	ServiceDemandStart ServiceStartType = wrappers.SERVICE_DEMAND_START
	ServiceDisabled    ServiceStartType = wrappers.SERVICE_DISABLED
)

type ServiceErrorControl uint32

const (
	ServiceErrorIgnore   ServiceErrorControl = wrappers.SERVICE_ERROR_IGNORE
	ServiceErrorNormal   ServiceErrorControl = wrappers.SERVICE_ERROR_NORMAL
	ServiceErrorSevere   ServiceErrorControl = wrappers.SERVICE_ERROR_SEVERE
	ServiceErrorCritical ServiceErrorControl = wrappers.SERVICE_ERROR_CRITICAL
)

type ServiceState uint32

const (
	ServiceStopped         ServiceState = wrappers.SERVICE_STOPPED
	ServiceStartPending    ServiceState = wrappers.SERVICE_START_PENDING
	ServiceStopPending     ServiceState = wrappers.SERVICE_STOP_PENDING
	ServiceRunning         ServiceState = wrappers.SERVICE_RUNNING
	ServiceContinuePending ServiceState = wrappers.SERVICE_CONTINUE_PENDING
	ServicePausePending    ServiceState = wrappers.SERVICE_PAUSE_PENDING
	ServicePaused          ServiceState = wrappers.SERVICE_PAUSED
)

type ServiceControlsAccepted uint32

const (
	ServiceAcceptStop                  ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_STOP
	ServiceAcceptPauseContinue         ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_PAUSE_CONTINUE
	ServiceAcceptShutdown              ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_SHUTDOWN
	ServiceAcceptParamChange           ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_PARAMCHANGE
	ServiceAcceptNetBindChange         ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_NETBINDCHANGE
	ServiceAcceptHardwareProfileChange ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_HARDWAREPROFILECHANGE
	ServiceAcceptPowerEvent            ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_POWEREVENT
	ServiceAcceptSessionChange         ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_SESSIONCHANGE
	ServiceAcceptPreshutdown           ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_PRESHUTDOWN
	ServiceAcceptTimeChange            ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_TIMECHANGE
	ServiceAcceptTriggerEvent          ServiceControlsAccepted = wrappers.SERVICE_ACCEPT_TRIGGEREVENT
)

type ServiceControl uint32

const (
	ServiceControlStop                  ServiceControl = wrappers.SERVICE_CONTROL_STOP
	ServiceControlPause                 ServiceControl = wrappers.SERVICE_CONTROL_PAUSE
	ServiceControlContinue              ServiceControl = wrappers.SERVICE_CONTROL_CONTINUE
	ServiceControlInterrogate           ServiceControl = wrappers.SERVICE_CONTROL_INTERROGATE
	ServiceControlShutdown              ServiceControl = wrappers.SERVICE_CONTROL_SHUTDOWN
	ServiceControlParamChange           ServiceControl = wrappers.SERVICE_CONTROL_PARAMCHANGE
	ServiceControlNetBindAdd            ServiceControl = wrappers.SERVICE_CONTROL_NETBINDADD
	ServiceControlNetBindRemove         ServiceControl = wrappers.SERVICE_CONTROL_NETBINDREMOVE
	ServiceControlNetBindEnable         ServiceControl = wrappers.SERVICE_CONTROL_NETBINDENABLE
	ServiceControlNetBindDisable        ServiceControl = wrappers.SERVICE_CONTROL_NETBINDDISABLE
	ServiceControlDeviceEvent           ServiceControl = wrappers.SERVICE_CONTROL_DEVICEEVENT
	ServiceControlHardwareProfileChange ServiceControl = wrappers.SERVICE_CONTROL_HARDWAREPROFILECHANGE
	ServiceControlPowerEvent            ServiceControl = wrappers.SERVICE_CONTROL_POWEREVENT
	ServiceControlSessionChange         ServiceControl = wrappers.SERVICE_CONTROL_SESSIONCHANGE
	ServiceControlPreshutdown           ServiceControl = wrappers.SERVICE_CONTROL_PRESHUTDOWN
	ServiceControlTimeChange            ServiceControl = wrappers.SERVICE_CONTROL_TIMECHANGE
	ServiceControlTriggerEvent          ServiceControl = wrappers.SERVICE_CONTROL_TRIGGEREVENT
)

type ServiceEnumState uint32

const (
	ServiceEnumActive   ServiceEnumState = wrappers.SERVICE_ACTIVE
	ServiceEnumInactive ServiceEnumState = wrappers.SERVICE_INACTIVE
	ServiceEnumAll      ServiceEnumState = wrappers.SERVICE_STATE_ALL
)

type ServiceConfigMask uint32

const (
	ServiceConfigServiceType      ServiceConfigMask = 0x00000001
	ServiceConfigStartType        ServiceConfigMask = 0x00000002
	ServiceConfigErrorControl     ServiceConfigMask = 0x00000004
	ServiceConfigBinaryPathName   ServiceConfigMask = 0x00000008
	ServiceConfigLoadOrderGroup   ServiceConfigMask = 0x00000010
	ServiceConfigTagId            ServiceConfigMask = 0x00000020
	ServiceConfigDependencies     ServiceConfigMask = 0x00000040
	ServiceConfigServiceStartName ServiceConfigMask = 0x00000080
	ServiceConfigPassword         ServiceConfigMask = 0x00000100
	ServiceConfigDisplayName      ServiceConfigMask = 0x00000200
)

type ServiceConfig struct {
	ServiceType      ServiceType
	StartType        ServiceStartType
	ErrorControl     ServiceErrorControl
	BinaryPathName   string
	LoadOrderGroup   string
	TagId            uint32
	Dependencies     string
	ServiceStartName string
	Password         string
	DisplayName      string
}

type ServiceStatusInfo struct {
	ServiceType             ServiceType
	CurrentState            ServiceState
	ControlsAccepted        ServiceControlsAccepted
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	CheckPoint              uint32
	WaitHint                uint32
}

type ServiceInfo struct {
	ServiceName   string
	DisplayName   string
	ServiceStatus ServiceStatusInfo
}

type Service struct {
	handle syscall.Handle
}

func (self *Service) Close() error {
	if self.handle != 0 {
		if err := wrappers.CloseServiceHandle(self.handle); err != nil {
			return NewWindowsError("CloseServiceHandle", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *Service) Control(control ServiceControl) (*ServiceStatusInfo, error) {
	var status wrappers.SERVICE_STATUS
	if err := wrappers.ControlService(self.handle, uint32(control), &status); err != nil {
		return nil, NewWindowsError("ControlService", err)
	}
	return &ServiceStatusInfo{
		ServiceType:             ServiceType(status.ServiceType),
		CurrentState:            ServiceState(status.CurrentState),
		ControlsAccepted:        ServiceControlsAccepted(status.ControlsAccepted),
		Win32ExitCode:           status.Win32ExitCode,
		ServiceSpecificExitCode: status.ServiceSpecificExitCode,
		CheckPoint:              status.CheckPoint,
		WaitHint:                status.WaitHint,
	}, nil
}

func (self *Service) Delete() error {
	if err := wrappers.DeleteService(self.handle); err != nil {
		return NewWindowsError("DeleteService", err)
	}
	return nil
}

func (self *Service) GetConfig() (*ServiceConfig, error) {
	var bytesNeeded uint32
	if err := wrappers.QueryServiceConfig(self.handle, nil, 0, &bytesNeeded); err != wrappers.ERROR_INSUFFICIENT_BUFFER {
		return nil, NewWindowsError("QueryServiceConfig", err)
	}
	buf := make([]byte, bytesNeeded)
	config := (*wrappers.QUERY_SERVICE_CONFIG)(unsafe.Pointer(&buf[0]))
	if err := wrappers.QueryServiceConfig(self.handle, config, bytesNeeded, &bytesNeeded); err != nil {
		return nil, NewWindowsError("QueryServiceConfig", err)
	}
	return &ServiceConfig{
		ServiceType:      ServiceType(config.ServiceType),
		StartType:        ServiceStartType(config.StartType),
		ErrorControl:     ServiceErrorControl(config.ErrorControl),
		BinaryPathName:   LpstrToString(config.BinaryPathName),
		LoadOrderGroup:   LpstrToString(config.LoadOrderGroup),
		TagId:            config.TagId,
		Dependencies:     LpstrToString(config.Dependencies),
		ServiceStartName: LpstrToString(config.ServiceStartName),
		DisplayName:      LpstrToString(config.DisplayName),
	}, nil
}

func (self *Service) GetDescription() (string, error) {
	var bytesNeeded uint32
	err := wrappers.QueryServiceConfig2(self.handle, wrappers.SERVICE_CONFIG_DESCRIPTION, nil, 0, &bytesNeeded)
	if err != wrappers.ERROR_INSUFFICIENT_BUFFER {
		return "", NewWindowsError("QueryServiceConfig2", err)
	}
	buf := make([]byte, bytesNeeded)
	err = wrappers.QueryServiceConfig2(
		self.handle,
		wrappers.SERVICE_CONFIG_DESCRIPTION,
		&buf[0],
		bytesNeeded,
		&bytesNeeded)
	if err != nil {
		return "", NewWindowsError("QueryServiceConfig2", err)
	}
	desc := (*wrappers.SERVICE_DESCRIPTION)(unsafe.Pointer(&buf[0]))
	return LpstrToString(desc.Description), nil
}

func (self *Service) GetProcessID() (uint, error) {
	var status wrappers.SERVICE_STATUS_PROCESS
	size := uint32(unsafe.Sizeof(status))
	err := wrappers.QueryServiceStatusEx(
		self.handle,
		wrappers.SC_STATUS_PROCESS_INFO,
		(*byte)(unsafe.Pointer(&status)),
		size,
		&size)
	if err != nil {
		return 0, NewWindowsError("QueryServiceStatusEx", err)
	}
	return uint(status.ProcessId), nil
}

func (self *Service) GetStatus() (*ServiceStatusInfo, error) {
	var status wrappers.SERVICE_STATUS
	if err := wrappers.QueryServiceStatus(self.handle, &status); err != nil {
		return nil, NewWindowsError("QueryServiceStatus", err)
	}
	return &ServiceStatusInfo{
		ServiceType:             ServiceType(status.ServiceType),
		CurrentState:            ServiceState(status.CurrentState),
		ControlsAccepted:        ServiceControlsAccepted(status.ControlsAccepted),
		Win32ExitCode:           status.Win32ExitCode,
		ServiceSpecificExitCode: status.ServiceSpecificExitCode,
		CheckPoint:              status.CheckPoint,
		WaitHint:                status.WaitHint,
	}, nil
}

func (self *Service) SetConfig(config *ServiceConfig, mask ServiceConfigMask) error {
	var serviceType uint32
	if (mask & ServiceConfigServiceType) != 0 {
		serviceType = uint32(config.ServiceType)
	} else {
		serviceType = wrappers.SERVICE_NO_CHANGE
	}
	var startType uint32
	if (mask & ServiceConfigStartType) != 0 {
		startType = uint32(config.StartType)
	} else {
		startType = wrappers.SERVICE_NO_CHANGE
	}
	var errorControl uint32
	if (mask & ServiceConfigErrorControl) != 0 {
		errorControl = uint32(config.ErrorControl)
	} else {
		errorControl = wrappers.SERVICE_NO_CHANGE
	}
	var binaryPathName *uint16
	if (mask & ServiceConfigBinaryPathName) != 0 {
		binaryPathName = syscall.StringToUTF16Ptr(config.BinaryPathName)
	}
	var loadOrderGroup *uint16
	if (mask & ServiceConfigLoadOrderGroup) != 0 {
		loadOrderGroup = syscall.StringToUTF16Ptr(config.LoadOrderGroup)
	}
	var tagId *uint32
	if (mask & ServiceConfigTagId) != 0 {
		tagId = &config.TagId
	}
	var dependencies *uint16
	if (mask & ServiceConfigDependencies) != 0 {
		dependencies = syscall.StringToUTF16Ptr(config.Dependencies)
	}
	var serviceStartName *uint16
	if (mask & ServiceConfigServiceStartName) != 0 {
		serviceStartName = syscall.StringToUTF16Ptr(config.ServiceStartName)
	}
	var password *uint16
	if (mask & ServiceConfigPassword) != 0 {
		password = syscall.StringToUTF16Ptr(config.Password)
	}
	var displayName *uint16
	if (mask & ServiceConfigDisplayName) != 0 {
		displayName = syscall.StringToUTF16Ptr(config.DisplayName)
	}
	err := wrappers.ChangeServiceConfig(
		self.handle,
		serviceType,
		startType,
		errorControl,
		binaryPathName,
		loadOrderGroup,
		tagId,
		dependencies,
		serviceStartName,
		password,
		displayName)
	if err != nil {
		return NewWindowsError("ChangeServiceConfig", err)
	}
	return nil
}

func (self *Service) SetDescription(description string) error {
	info := &wrappers.SERVICE_DESCRIPTION{Description: syscall.StringToUTF16Ptr(description)}
	err := wrappers.ChangeServiceConfig2(self.handle, wrappers.SERVICE_CONFIG_DESCRIPTION, (*byte)(unsafe.Pointer(info)))
	if err != nil {
		return NewWindowsError("ChangeServiceConfig2", err)
	}
	return nil
}

func (self *Service) Start(args []string) error {
	var argVectors **uint16
	if len(args) > 0 {
		argSlice := make([]*uint16, len(args))
		for i, arg := range args {
			argSlice[i] = syscall.StringToUTF16Ptr(arg)
		}
		argVectors = &argSlice[0]
	}
	if err := wrappers.StartService(self.handle, uint32(len(args)), argVectors); err != nil {
		return NewWindowsError("StartService", err)
	}
	return nil
}

type SCManager struct {
	handle syscall.Handle
}

func OpenLocalSCManager() (*SCManager, error) {
	handle, err := wrappers.OpenSCManager(
		nil,
		syscall.StringToUTF16Ptr(wrappers.SERVICES_ACTIVE_DATABASE),
		wrappers.SC_MANAGER_ALL_ACCESS)
	if err != nil {
		return nil, NewWindowsError("OpenSCManager", err)
	}
	return &SCManager{handle: handle}, nil
}

func (self *SCManager) Close() error {
	if self.handle != 0 {
		if err := wrappers.CloseServiceHandle(self.handle); err != nil {
			return NewWindowsError("CloseServiceHandle", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *SCManager) CreateService(serviceName string, config *ServiceConfig, mask ServiceConfigMask) (*Service, error) {
	var binaryPathName *uint16
	if (mask & ServiceConfigBinaryPathName) != 0 {
		binaryPathName = syscall.StringToUTF16Ptr(config.BinaryPathName)
	}
	var loadOrderGroup *uint16
	if (mask & ServiceConfigLoadOrderGroup) != 0 {
		loadOrderGroup = syscall.StringToUTF16Ptr(config.LoadOrderGroup)
	}
	var tagId *uint32
	if (mask & ServiceConfigTagId) != 0 {
		tagId = &config.TagId
	}
	var dependencies *uint16
	if (mask & ServiceConfigDependencies) != 0 {
		dependencies = syscall.StringToUTF16Ptr(config.Dependencies)
	}
	var serviceStartName *uint16
	if (mask & ServiceConfigServiceStartName) != 0 {
		serviceStartName = syscall.StringToUTF16Ptr(config.ServiceStartName)
	}
	var password *uint16
	if (mask & ServiceConfigPassword) != 0 {
		password = syscall.StringToUTF16Ptr(config.Password)
	}
	var displayName *uint16
	if (mask & ServiceConfigDisplayName) != 0 {
		displayName = syscall.StringToUTF16Ptr(config.DisplayName)
	}
	handle, err := wrappers.CreateService(
		self.handle,
		syscall.StringToUTF16Ptr(serviceName),
		displayName,
		wrappers.SERVICE_ALL_ACCESS,
		uint32(config.ServiceType),
		uint32(config.StartType),
		uint32(config.ErrorControl),
		binaryPathName,
		loadOrderGroup,
		tagId,
		dependencies,
		serviceStartName,
		password)
	if err != nil {
		return nil, NewWindowsError("CreateService", err)
	}
	return &Service{handle: handle}, nil
}

func (self *SCManager) GetServices(serviceType ServiceType, serviceState ServiceEnumState) ([]ServiceInfo, error) {
	services := []ServiceInfo{}
	hasMore := true
	var resumeHandle uint32
	for hasMore {
		var bytesNeeded uint32
		var servicesReturned uint32
		err := wrappers.EnumServicesStatus(
			self.handle,
			uint32(serviceType),
			uint32(serviceState),
			nil,
			0,
			&bytesNeeded,
			&servicesReturned,
			&resumeHandle)
		if err != wrappers.ERROR_INSUFFICIENT_BUFFER && err != wrappers.ERROR_MORE_DATA {
			return nil, NewWindowsError("EnumServicesStatus", err)
		}
		buf := make([]byte, bytesNeeded)
		err = wrappers.EnumServicesStatus(
			self.handle,
			uint32(serviceType),
			uint32(serviceState),
			&buf[0],
			bytesNeeded,
			&bytesNeeded,
			&servicesReturned,
			&resumeHandle)
		if err == nil {
			hasMore = false
		} else if err != wrappers.ERROR_MORE_DATA {
			return nil, NewWindowsError("EnumServicesStatus", err)
		}
		dataSize := int(unsafe.Sizeof(wrappers.ENUM_SERVICE_STATUS{}))
		for i := 0; i < int(servicesReturned); i++ {
			data := (*wrappers.ENUM_SERVICE_STATUS)(unsafe.Pointer(&buf[i*dataSize]))
			services = append(services, ServiceInfo{
				ServiceName:   LpstrToString(data.ServiceName),
				DisplayName:   LpstrToString(data.DisplayName),
				ServiceStatus: ServiceStatusInfo{
					ServiceType:             ServiceType(data.ServiceStatus.ServiceType),
					CurrentState:            ServiceState(data.ServiceStatus.CurrentState),
					ControlsAccepted:        ServiceControlsAccepted(data.ServiceStatus.ControlsAccepted),
					Win32ExitCode:           data.ServiceStatus.Win32ExitCode,
					ServiceSpecificExitCode: data.ServiceStatus.ServiceSpecificExitCode,
					CheckPoint:              data.ServiceStatus.CheckPoint,
					WaitHint:                data.ServiceStatus.WaitHint,
				},
			})
		}
	}
	return services, nil
}

func (self *SCManager) OpenService(serviceName string) (*Service, error) {
	handle, err := wrappers.OpenService(
		self.handle,
		syscall.StringToUTF16Ptr(serviceName),
		wrappers.SERVICE_ALL_ACCESS)
	if err != nil {
		return nil, NewWindowsError("OpenService", err)
	}
	return &Service{handle: handle}, nil
}

func IsServiceRunning(serviceName string) (bool, error) {
	// This function requires fewer access rights than using the above classes and can be used to check if a service
	// is running without administrator access.
	scmhandle, err := wrappers.OpenSCManager(
		nil,
		syscall.StringToUTF16Ptr(wrappers.SERVICES_ACTIVE_DATABASE),
		wrappers.SC_MANAGER_CONNECT)
	if err != nil {
		return false, NewWindowsError("OpenSCManager", err)
	}
	defer wrappers.CloseServiceHandle(scmhandle)
	handle, err := wrappers.OpenService(
		scmhandle,
		syscall.StringToUTF16Ptr(serviceName),
		wrappers.SERVICE_QUERY_STATUS)
	if err == wrappers.ERROR_SERVICE_DOES_NOT_EXIST {
		return false, nil
	} else if err != nil {
		return false, NewWindowsError("OpenService", err)
	}
	defer wrappers.CloseServiceHandle(handle)
	var status wrappers.SERVICE_STATUS
	if err := wrappers.QueryServiceStatus(handle, &status); err != nil {
		return false, NewWindowsError("QueryServiceStatus", err)
	}
	return status.CurrentState == wrappers.SERVICE_RUNNING, nil
}
