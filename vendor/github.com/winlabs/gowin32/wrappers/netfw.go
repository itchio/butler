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

var (
	IID_INetFwRule     = GUID{0xAF230D27, 0xBABA, 0x4E42, [8]byte{0xAC, 0xED, 0xF5, 0x24, 0xF2, 0x2C, 0xFC, 0xE2}}
	IID_INetFwRules    = GUID{0x9C4C6277, 0x5027, 0x441E, [8]byte{0xAF, 0xAE, 0xCA, 0x1F, 0x54, 0x2D, 0xA0, 0x09}}
	IID_INetFwPolicy2  = GUID{0x98325047, 0xC671, 0x4174, [8]byte{0x8D, 0x81, 0xDE, 0xFC, 0xD3, 0xF0, 0x31, 0x86}}
	IID_INetFwMgr      = GUID{0xF7898AF5, 0xCAC4, 0x4632, [8]byte{0xA2, 0xEC, 0xDA, 0x06, 0xE5, 0x11, 0x1A, 0xF2}}
	CLSID_NetFwRule    = GUID{0x2C5BC43E, 0x3369, 0x4C33, [8]byte{0xAB, 0x0C, 0xBE, 0x94, 0x69, 0x67, 0x7A, 0xF4}}
	CLSID_NetFwPolicy2 = GUID{0xE2B3C97F, 0x6AE1, 0x41AC, [8]byte{0x81, 0x7A, 0xF6, 0xF9, 0x21, 0x66, 0xD7, 0xDD}}
	CLSID_NetFwMgr     = GUID{0x304CE942, 0x6E39, 0x40D8, [8]byte{0x94, 0x3A, 0xB9, 0x13, 0xC4, 0x0C, 0x9C, 0xD4}}
)

type INetFwRuleVtbl struct {
	IDispatchVtbl
	Get_Name              uintptr
	Put_Name              uintptr
	Get_Description       uintptr
	Put_Description       uintptr
	Get_ApplicationName   uintptr
	Put_ApplicationName   uintptr
	Get_ServiceName       uintptr
	Put_ServiceName       uintptr
	Get_Protocol          uintptr
	Put_Protocol          uintptr
	Get_LocalPorts        uintptr
	Put_LocalPorts        uintptr
	Get_RemotePorts       uintptr
	Put_RemotePorts       uintptr
	Get_LocalAddresses    uintptr
	Put_LocalAddresses    uintptr
	Get_RemoteAddresses   uintptr
	Put_RemoteAddresses   uintptr
	Get_IcmpTypesAndCodes uintptr
	Put_IcmpTypesAndCodes uintptr
	Get_Direction         uintptr
	Put_Direction         uintptr
	Get_Interfaces        uintptr
	Put_Interfaces        uintptr
	Get_InterfaceTypes    uintptr
	Put_InterfaceTypes    uintptr
	Get_Enabled           uintptr
	Put_Enabled           uintptr
	Get_Grouping          uintptr
	Put_Grouping          uintptr
	Get_Profiles          uintptr
	Put_Profiles          uintptr
	Get_EdgeTraversal     uintptr
	Put_EdgeTraversal     uintptr
	Get_Action            uintptr
	Put_Action            uintptr
}

type INetFwRule struct {
	IDispatch
}

func (self *INetFwRule) Get_Name(name **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Name,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(name)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Name(name *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Name,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(name)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Description(desc **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Description,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(desc)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Description(desc *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Description,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(desc)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_ApplicationName(imageFileName **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_ApplicationName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(imageFileName)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_ApplicationName(imageFileName *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_ApplicationName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(imageFileName)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_ServiceName(serviceName **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_ServiceName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(serviceName)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_ServiceName(serviceName *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_ServiceName,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(serviceName)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Protocol(protocol *int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Protocol,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(protocol)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Protocol(protocol int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Protocol,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(protocol),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_LocalPorts(portNumbers **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_LocalPorts,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(portNumbers)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_LocalPorts(portNumbers *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_LocalPorts,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(portNumbers)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_RemotePorts(portNumbers **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_RemotePorts,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(portNumbers)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_RemotePorts(portNumbers *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_RemotePorts,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(portNumbers)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_LocalAddresses(localAddrs **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_LocalAddresses,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(localAddrs)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_LocalAddresses(localAddrs *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_LocalAddresses,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(localAddrs)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_RemoteAddresses(remoteAddrs **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_RemoteAddresses,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(remoteAddrs)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_RemoteAddresses(remoteAddrs *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_RemoteAddresses,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(remoteAddrs)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_IcmpTypesAndCodes(icmpTypesAndCodes **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_IcmpTypesAndCodes,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(icmpTypesAndCodes)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_IcmpTypesAndCodes(icmpTypesAndCodes *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_IcmpTypesAndCodes,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(icmpTypesAndCodes)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Direction(dir *int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Direction,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(dir)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Direction(dir int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Direction,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(dir),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_InterfaceTypes(interfaceTypes **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_InterfaceTypes,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(interfaceTypes)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_InterfaceTypes(interfaceTypes *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_InterfaceTypes,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(interfaceTypes)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Enabled(enabled *bool) uint32 {
	if enabled == nil {
		return E_POINTER
	}
	enabledRaw := int16(VARIANT_FALSE)
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Enabled,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(&enabledRaw)),
		0)
	*enabled = (enabledRaw != VARIANT_FALSE)
	return uint32(r1)
}

func (self *INetFwRule) Put_Enabled(enabled bool) uint32 {
	var enabledRaw int16
	if enabled {
		enabledRaw = VARIANT_TRUE
	} else {
		enabledRaw = VARIANT_FALSE
	}
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Enabled,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(enabledRaw),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Grouping(context **uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Grouping,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(context)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Grouping(context *uint16) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Grouping,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(context)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Profiles(profileTypesBitmask *int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Profiles,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(profileTypesBitmask)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Profiles(profileTypesBitmask int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Profiles,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(profileTypesBitmask),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_EdgeTraversal(enabled *bool) uint32 {
	if enabled == nil {
		return E_POINTER
	}
	enabledRaw := int16(VARIANT_FALSE)
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_EdgeTraversal,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(&enabledRaw)),
		0)
	*enabled = (enabledRaw != VARIANT_FALSE)
	return uint32(r1)
}

func (self *INetFwRule) Put_EdgeTraversal(enabled bool) uint32 {
	var enabledRaw int16
	if enabled {
		enabledRaw = VARIANT_TRUE
	} else {
		enabledRaw = VARIANT_FALSE
	}
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_EdgeTraversal,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(enabledRaw),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Get_Action(action *int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Action,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(action)),
		0)
	return uint32(r1)
}

func (self *INetFwRule) Put_Action(action int32) uint32 {
	vtbl := (*INetFwRuleVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Put_Action,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(action),
		0)
	return uint32(r1)
}

type INetFwRulesVtbl struct {
	IDispatchVtbl
	Get_Count    uintptr
	Add          uintptr
	Remove       uintptr
	Item         uintptr
	Get__NewEnum uintptr
}

type INetFwRules struct {
	IDispatch
}

func (self *INetFwRules) Get_Count(count *int32) uint32 {
	vtbl := (*INetFwRulesVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Count,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(count)),
		0)
	return uint32(r1)
}

func (self *INetFwRules) Add(rule *INetFwRule) uint32 {
	vtbl := (*INetFwRulesVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Add,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(rule)),
		0)
	return uint32(r1)
}

func (self *INetFwRules) Remove(name *uint16) uint32 {
	vtbl := (*INetFwRulesVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Remove,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(name)),
		0)
	return uint32(r1)
}

func (self *INetFwRules) Item(name *uint16, rule **INetFwRule) uint32 {
	vtbl := (*INetFwRulesVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Item,
		3,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(rule)))
	return uint32(r1)
}

func (self *INetFwRules) Get__NewEnum(newEnum **IUnknown) uint32 {
	vtbl := (*INetFwRulesVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get__NewEnum,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(newEnum)),
		0)
	return uint32(r1)
}

type INetFwPolicy2Vtbl struct {
	IDispatchVtbl
	Get_CurrentProfileTypes                          uintptr
	Get_FirewallEnabled                              uintptr
	Put_FirewallEnabled                              uintptr
	Get_ExcludedInterfaces                           uintptr
	Put_ExcludedInterfaces                           uintptr
	Get_BlockAllInboundTraffic                       uintptr
	Put_BlockAllInboundTraffic                       uintptr
	Get_NotificationsDisabled                        uintptr
	Put_NotificationsDisabled                        uintptr
	Get_UnicastResponsesToMulticastBroadcastDisabled uintptr
	Put_UnicastRepsonsesToMulticastBroadcastDisabled uintptr
	Get_Rules                                        uintptr
	Get_ServiceRestriction                           uintptr
	EnableRuleGroup                                  uintptr
	IsRuleGroupEnabled                               uintptr
	RestoreLocalFirewallDefaults                     uintptr
	Get_DefaultInboundAction                         uintptr
	Put_DefaultInboundAction                         uintptr
	Get_DefaultOutboundAction                        uintptr
	Put_DefaultOutboundAction                        uintptr
	Get_IsRuleGroupCurrentlyEnabled                  uintptr
	Get_LocalPolicyModifyState                       uintptr
}

type INetFwPolicy2 struct {
	IDispatch
}

func (self *INetFwPolicy2) Get_Rules(rules **INetFwRules) uint32 {
	vtbl := (*INetFwPolicy2Vtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_Rules,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(rules)),
		0)
	return uint32(r1)
}

type INetFwMgrVtbl struct {
	IDispatchVtbl
	Get_LocalPolicy        uintptr
	Get_CurrentProfileType uintptr
	RestoreDefaults        uintptr
	IsPortAllowed          uintptr
	IsIcmpTypeAllowed      uintptr
}

type INetFwMgr struct {
	IDispatch
}

func (self *INetFwMgr) Get_CurrentProfileType(profileType *int32) uint32 {
	vtbl := (*INetFwMgrVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(
		vtbl.Get_CurrentProfileType,
		2,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(profileType)),
		0)
	return uint32(r1)
}

func (self *INetFwMgr) RestoreDefaults() uint32 {
	vtbl := (*INetFwMgrVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall(vtbl.RestoreDefaults, 1, uintptr(unsafe.Pointer(self)), 0, 0)
	return uint32(r1)
}

func (self *INetFwMgr) IsPortAllowed(imageFileName *uint16, ipVersion int32, portNumber int32, localAddress *uint16, ipProtocol int32, allowed *VARIANT, restricted *VARIANT) uint32 {
	vtbl := (*INetFwMgrVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall9(
		vtbl.IsPortAllowed,
		8,
		uintptr(unsafe.Pointer(self)),
		uintptr(unsafe.Pointer(imageFileName)),
		uintptr(ipVersion),
		uintptr(portNumber),
		uintptr(unsafe.Pointer(localAddress)),
		uintptr(ipProtocol),
		uintptr(unsafe.Pointer(allowed)),
		uintptr(unsafe.Pointer(restricted)),
		0)
	return uint32(r1)
}

func (self *INetFwMgr) IsIcmpTypeAllowed(ipVersion int32, localAddress *uint16, icmpType *uint16, allowed *VARIANT, restricted *VARIANT) uint32 {
	vtbl := (*INetFwMgrVtbl)(unsafe.Pointer(self.Vtbl))
	r1, _, _ := syscall.Syscall6(
		vtbl.IsIcmpTypeAllowed,
		6,
		uintptr(unsafe.Pointer(self)),
		uintptr(ipVersion),
		uintptr(unsafe.Pointer(localAddress)),
		uintptr(unsafe.Pointer(icmpType)),
		uintptr(unsafe.Pointer(allowed)),
		uintptr(unsafe.Pointer(restricted)))
	return uint32(r1)
}
