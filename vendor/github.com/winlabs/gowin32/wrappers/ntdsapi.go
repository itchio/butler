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
	DS_SPN_DNS_HOST  = 0
	DS_SPN_DN_HOST   = 1
	DS_SPN_NB_HOST   = 2
	DS_SPN_DOMAIN    = 3
	DS_SPN_NB_DOMAIN = 4
	DS_SPN_SERVICE   = 5
)

const (
	DS_SPN_ADD_SPN_OP     = 0
	DS_SPN_REPLACE_SPN_OP = 1
	DS_SPN_DELETE_SPN_OP  = 2
)

var (
	modntdsapi = syscall.NewLazyDLL("ntdsapi.dll")

	procDsBindW              = modntdsapi.NewProc("DsBindW")
	procDsFreeSpnArrayW      = modntdsapi.NewProc("DsFreeSpnArrayW")
	procDsGetSpnW            = modntdsapi.NewProc("DsGetSpnW")
	procDsMakeSpnW           = modntdsapi.NewProc("DsMakeSpnW")
	procDsServerRegisterSpnW = modntdsapi.NewProc("DsServerRegisterSpnW")
	procDsUnBindW            = modntdsapi.NewProc("DsUnBindW")
	procDsWriteAccountSpnW   = modntdsapi.NewProc("DsWriteAccountSpnW")
)

func DsBind(domainControllerName *uint16, dnsDomainName *uint16, hDS *syscall.Handle) error {
	r1, _, _ := syscall.Syscall(
		procDsBindW.Addr(),
		3,
		uintptr(unsafe.Pointer(domainControllerName)),
		uintptr(unsafe.Pointer(dnsDomainName)),
		uintptr(unsafe.Pointer(hDS)))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func DsFreeSpnArray(cSpn uint32, spn **uint16) {
	syscall.Syscall(
		procDsFreeSpnArrayW.Addr(),
		2,
		uintptr(cSpn),
		uintptr(unsafe.Pointer(spn)),
		0)
}

func DsGetSpn(serviceType int32, serviceClass *uint16, serviceName *uint16, instancePort uint16, cInstanceNames uint16, instanceNames **uint16, instancePorts *uint16, cSpn *uint32, spn ***uint16) error {
	r1, _, _ := syscall.Syscall9(
		procDsGetSpnW.Addr(),
		9,
		uintptr(serviceType),
		uintptr(unsafe.Pointer(serviceClass)),
		uintptr(unsafe.Pointer(serviceName)),
		uintptr(instancePort),
		uintptr(cInstanceNames),
		uintptr(unsafe.Pointer(instanceNames)),
		uintptr(unsafe.Pointer(instancePorts)),
		uintptr(unsafe.Pointer(cSpn)),
		uintptr(unsafe.Pointer(spn)))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func DsMakeSpn(serviceClass *uint16, serviceName *uint16, instanceName *uint16, instancePort uint16, referrer *uint16, spnLength *uint32, spn *uint16) error {
	r1, _, _ := syscall.Syscall9(
		procDsMakeSpnW.Addr(),
		7,
		uintptr(unsafe.Pointer(serviceClass)),
		uintptr(unsafe.Pointer(serviceName)),
		uintptr(unsafe.Pointer(instanceName)),
		uintptr(instancePort),
		uintptr(unsafe.Pointer(referrer)),
		uintptr(unsafe.Pointer(spnLength)),
		uintptr(unsafe.Pointer(spn)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func DsServerRegisterSpn(operation int32, serviceClass *uint16, userObjectDN *uint16) error {
	r1, _, _ := syscall.Syscall(
		procDsServerRegisterSpnW.Addr(),
		3,
		uintptr(operation),
		uintptr(unsafe.Pointer(serviceClass)),
		uintptr(unsafe.Pointer(userObjectDN)))
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func DsUnBind(hDS *syscall.Handle) error {
	r1, _, _ := syscall.Syscall(
		procDsUnBindW.Addr(),
		1,
		uintptr(unsafe.Pointer(hDS)),
		0,
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}

func DsWriteAccountSpn(hDS syscall.Handle, operation int32, account *uint16, cSpn uint32, spn *uint16) error {
	r1, _, _ := syscall.Syscall6(
		procDsWriteAccountSpnW.Addr(),
		5,
		uintptr(hDS),
		uintptr(operation),
		uintptr(unsafe.Pointer(account)),
		uintptr(cSpn),
		uintptr(unsafe.Pointer(spn)),
		0)
	if err := syscall.Errno(r1); err != ERROR_SUCCESS {
		return err
	}
	return nil
}
