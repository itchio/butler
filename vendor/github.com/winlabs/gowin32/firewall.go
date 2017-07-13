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

type FirewallIPVersion int32

const (
	FirewallIPV4         FirewallIPVersion = wrappers.NET_FW_IP_VERSION_V4
	FirewallIPV6         FirewallIPVersion = wrappers.NET_FW_IP_VERSION_V6
	FirewallAnyIPVersion FirewallIPVersion = wrappers.NET_FW_IP_VERSION_ANY
)

type FirewallProtocol int32

const (
	FirewallTCP         FirewallProtocol = wrappers.NET_FW_PROTOCOL_TCP
	FirewallUDP         FirewallProtocol = wrappers.NET_FW_PROTOCOL_UDP
	FirewallAnyProtocol FirewallProtocol = wrappers.NET_FW_PROTOCOL_ANY
)

type FirewallDirection int32

const (
	FirewallInbound  FirewallDirection = wrappers.NET_FW_RULE_DIR_IN
	FirewallOutbound FirewallDirection = wrappers.NET_FW_RULE_DIR_OUT
)

type FirewallAction int32

const (
	FirewallBlock FirewallAction = wrappers.NET_FW_ACTION_BLOCK
	FirewallAllow FirewallAction = wrappers.NET_FW_ACTION_ALLOW
)

type FirewallRule struct {
	object *wrappers.INetFwRule
}

func NewFirewallRule() (*FirewallRule, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_NetFwRule,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_INetFwRule,
		&object)
	if wrappers.FAILED(hr) {
		return nil, NewWindowsError("CoCreateInstance", COMError(hr))
	}
	return &FirewallRule{object: (*wrappers.INetFwRule)(unsafe.Pointer(object))}, nil
}

func (self *FirewallRule) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *FirewallRule) GetName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_Name", COMErrorPointer)
	}
	var nameRaw *uint16
	if hr := self.object.Get_Name(&nameRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_Name", COMError(hr))
	}
	return BstrToString(nameRaw), nil
}

func (self *FirewallRule) SetName(name string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Name", COMErrorPointer)
	}
	nameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(name))
	defer wrappers.SysFreeString(nameRaw)
	if hr := self.object.Put_Name(nameRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Name", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetDescription() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_Description", COMErrorPointer)
	}
	var descRaw *uint16
	if hr := self.object.Get_Description(&descRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_Description", COMError(hr))
	}
	return BstrToString(descRaw), nil
}

func (self *FirewallRule) SetDescription(desc string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Description", COMErrorPointer)
	}
	descRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(desc))
	defer wrappers.SysFreeString(descRaw)
	if hr := self.object.Put_Description(descRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Description", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetApplicationName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_ApplicationName", COMErrorPointer)
	}
	var imageFileNameRaw *uint16
	if hr := self.object.Get_ApplicationName(&imageFileNameRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_ApplicationName", COMError(hr))
	}
	return BstrToString(imageFileNameRaw), nil
}

func (self *FirewallRule) SetApplicationName(imageFileName string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_ApplicationName", COMErrorPointer)
	}
	imageFileNameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(imageFileName))
	defer wrappers.SysFreeString(imageFileNameRaw)
	if hr := self.object.Put_ApplicationName(imageFileNameRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_ApplicationName", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetServiceName() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_ServiceName", COMErrorPointer)
	}
	var serviceNameRaw *uint16
	if hr := self.object.Get_ServiceName(&serviceNameRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_ServiceName", COMError(hr))
	}
	return BstrToString(serviceNameRaw), nil
}

func (self *FirewallRule) SetServiceName(serviceName string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_ServiceName", COMErrorPointer)
	}
	serviceNameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(serviceName))
	defer wrappers.SysFreeString(serviceNameRaw)
	if hr := self.object.Put_ServiceName(serviceNameRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_ServiceName", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetProtocol() (FirewallProtocol, error) {
	if self.object == nil {
		return 0, NewWindowsError("INetFwRule::get_Protocol", COMErrorPointer)
	}
	var protocolRaw int32
	if hr := self.object.Get_Protocol(&protocolRaw); wrappers.FAILED(hr) {
		return 0, NewWindowsError("INetFwRule::get_Protocol", COMError(hr))
	}
	return FirewallProtocol(protocolRaw), nil
}

func (self *FirewallRule) SetProtocol(protocol FirewallProtocol) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Protocol", COMErrorPointer)
	}
	if hr := self.object.Put_Protocol(int32(protocol)); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Protocol", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetLocalPorts() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_LocalPorts", COMErrorPointer)
	}
	var portNumbersRaw *uint16
	if hr := self.object.Get_LocalPorts(&portNumbersRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_LocalPorts", COMError(hr))
	}
	return BstrToString(portNumbersRaw), nil
}

func (self *FirewallRule) SetLocalPorts(portNumbers string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_LocalPorts", COMErrorPointer)
	}
	portNumbersRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(portNumbers))
	defer wrappers.SysFreeString(portNumbersRaw)
	if hr := self.object.Put_LocalPorts(portNumbersRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_LocalPorts", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetRemotePorts() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_RemotePorts", COMErrorPointer)
	}
	var portNumbersRaw *uint16
	if hr := self.object.Get_RemotePorts(&portNumbersRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_RemotePorts", COMError(hr))
	}
	return BstrToString(portNumbersRaw), nil
}

func (self *FirewallRule) SetRemotePorts(portNumbers string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_RemotePorts", COMErrorPointer)
	}
	portNumbersRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(portNumbers))
	defer wrappers.SysFreeString(portNumbersRaw)
	if hr := self.object.Put_RemotePorts(portNumbersRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_RemotePorts", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetLocalAddresses() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_LocalAddresses", COMErrorPointer)
	}
	var localAddrsRaw *uint16
	if hr := self.object.Get_LocalAddresses(&localAddrsRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_LocalAddresses", COMError(hr))
	}
	return BstrToString(localAddrsRaw), nil
}

func (self *FirewallRule) SetLocalAddresses(localAddrs string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_LocalAddresses", COMErrorPointer)
	}
	localAddrsRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(localAddrs))
	defer wrappers.SysFreeString(localAddrsRaw)
	if hr := self.object.Put_LocalAddresses(localAddrsRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_LocalAddresses", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetRemoteAddresses() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_RemoteAddresses", COMErrorPointer)
	}
	var remoteAddrsRaw *uint16
	if hr := self.object.Get_RemoteAddresses(&remoteAddrsRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_RemoteAddresses", COMError(hr))
	}
	return BstrToString(remoteAddrsRaw), nil
}

func (self *FirewallRule) SetRemoteAddresses(remoteAddrs string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_RemoteAddresses", COMErrorPointer)
	}
	remoteAddrsRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(remoteAddrs))
	defer wrappers.SysFreeString(remoteAddrsRaw)
	if hr := self.object.Put_RemoteAddresses(remoteAddrsRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_RemoteAddresses", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetIcmpTypesAndCodes() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_IcmpTypesAndCodes", COMErrorPointer)
	}
	var icmpTypesAndCodesRaw *uint16
	if hr := self.object.Get_IcmpTypesAndCodes(&icmpTypesAndCodesRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_IcmpTypesAndCodes", COMError(hr))
	}
	return BstrToString(icmpTypesAndCodesRaw), nil
}

func (self *FirewallRule) SetIcmpTypesAndCodes(icmpTypesAndCodes string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_IcmpTypesAndCodes", COMErrorPointer)
	}
	icmpTypesAndCodesRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(icmpTypesAndCodes))
	defer wrappers.SysFreeString(icmpTypesAndCodesRaw)
	if hr := self.object.Put_IcmpTypesAndCodes(icmpTypesAndCodesRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_IcmpTypesAndCodes", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetDirection() (FirewallDirection, error) {
	if self.object == nil {
		return 0, NewWindowsError("INetFwRule::get_Direction", COMErrorPointer)
	}
	var dirRaw int32
	if hr := self.object.Get_Direction(&dirRaw); wrappers.FAILED(hr) {
		return 0, NewWindowsError("INetFwRule::get_Direction", COMError(hr))
	}
	return FirewallDirection(dirRaw), nil
}

func (self *FirewallRule) SetDirection(dir FirewallDirection) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Direction", COMErrorPointer)
	}
	if hr := self.object.Put_Direction(int32(dir)); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Direction", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetInterfaceTypes() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_InterfaceTypes", COMErrorPointer)
	}
	var interfaceTypesRaw *uint16
	if hr := self.object.Get_InterfaceTypes(&interfaceTypesRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_InterfaceTypes", COMError(hr))
	}
	return BstrToString(interfaceTypesRaw), nil
}

func (self *FirewallRule) SetInterfaceTypes(interfaceTypes string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_InterfaceTypes", COMErrorPointer)
	}
	interfaceTypesRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(interfaceTypes))
	defer wrappers.SysFreeString(interfaceTypesRaw)
	if hr := self.object.Put_InterfaceTypes(interfaceTypesRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_InterfaceTypes", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetEnabled() (bool, error) {
	if self.object == nil {
		return false, NewWindowsError("INetFwRule::get_Enabled", COMErrorPointer)
	}
	var enabled bool
	if hr := self.object.Get_Enabled(&enabled); wrappers.FAILED(hr) {
		return false, NewWindowsError("INetFwRule::get_Enabled", COMError(hr))
	}
	return enabled, nil
}

func (self *FirewallRule) SetEnabled(enabled bool) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Enabled", COMErrorPointer)
	}
	if hr := self.object.Put_Enabled(enabled); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Enabled", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetGrouping() (string, error) {
	if self.object == nil {
		return "", NewWindowsError("INetFwRule::get_Grouping", COMErrorPointer)
	}
	var contextRaw *uint16
	if hr := self.object.Get_Grouping(&contextRaw); wrappers.FAILED(hr) {
		return "", NewWindowsError("INetFwRule::get_Grouping", COMError(hr))
	}
	return BstrToString(contextRaw), nil
}

func (self *FirewallRule) SetGrouping(context string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Grouping", COMErrorPointer)
	}
	contextRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(context))
	defer wrappers.SysFreeString(contextRaw)
	if hr := self.object.Put_Grouping(contextRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_grouping", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetEdgeTraversal() (bool, error) {
	if self.object == nil {
		return false, NewWindowsError("INetFwRule::get_EdgeTraversal", COMErrorPointer)
	}
	var enabled bool
	if hr := self.object.Get_EdgeTraversal(&enabled); wrappers.FAILED(hr) {
		return false, NewWindowsError("INetFwRule::put_EdgeTraversal", COMError(hr))
	}
	return enabled, nil
}

func (self *FirewallRule) SetEdgeTraversal(enabled bool) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_EdgeTraversal", COMErrorPointer)
	}
	if hr := self.object.Put_EdgeTraversal(enabled); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_EdgeTraversal", COMError(hr))
	}
	return nil
}

func (self *FirewallRule) GetAction() (FirewallAction, error) {
	if self.object == nil {
		return 0, NewWindowsError("INetFwRule::get_Action", COMErrorPointer)
	}
	var actionRaw int32
	if hr := self.object.Get_Action(&actionRaw); wrappers.FAILED(hr) {
		return 0, NewWindowsError("INetFwRule::get_Action", COMError(hr))
	}
	return FirewallAction(actionRaw), nil
}

func (self *FirewallRule) SetAction(action FirewallAction) error {
	if self.object == nil {
		return NewWindowsError("INetFwRule::put_Action", COMErrorPointer)
	}
	if hr := self.object.Put_Action(int32(action)); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRule::put_Action", COMError(hr))
	}
	return nil
}

type FirewallRuleEnumerator struct {
	object *wrappers.IEnumVARIANT
}

func (self *FirewallRuleEnumerator) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *FirewallRuleEnumerator) Next() (*FirewallRule, error) {
	if self.object == nil {
		return nil, NewWindowsError("IEnumVARIANT::Next", COMErrorPointer)
	}
	var varRaw wrappers.VARIANT
	if hr := self.object.Next(1, &varRaw, nil); wrappers.FAILED(hr) {
		return nil, NewWindowsError("IEnumVARIANT::Next", COMError(hr))
	} else if hr == wrappers.S_FALSE {
		return nil, nil
	}
	defer wrappers.VariantClear(&varRaw)
	var varDispatch wrappers.VARIANT
	if hr := wrappers.VariantChangeType(&varDispatch, &varRaw, 0, wrappers.VT_DISPATCH); wrappers.FAILED(hr) {
		return nil, NewWindowsError("VariantChangeType", COMError(hr))
	}
	defer wrappers.VariantClear(&varDispatch)
	dispatch := (*wrappers.IDispatch)(unsafe.Pointer(uintptr(varDispatch.Val[0])))
	var rule uintptr
	if hr := dispatch.QueryInterface(&wrappers.IID_INetFwRule, &rule); wrappers.FAILED(hr) {
		return nil, NewWindowsError("IUnknown::QueryInterface", COMError(hr))
	}
	return &FirewallRule{object: (*wrappers.INetFwRule)(unsafe.Pointer(rule))}, nil
}

func (self *FirewallRuleEnumerator) Skip(count uint) error {
	if self.object == nil {
		return NewWindowsError("IEnumVARIANT::Skip", COMErrorPointer)
	}
	if hr := self.object.Skip(uint32(count)); wrappers.FAILED(hr) {
		return NewWindowsError("IEnumVARIANT::Skip", COMError(hr))
	}
	return nil
}

func (self *FirewallRuleEnumerator) Reset() error {
	if self.object == nil {
		return NewWindowsError("IEnumVARIANT::Reset", COMErrorPointer)
	}
	if hr := self.object.Reset(); wrappers.FAILED(hr) {
		return NewWindowsError("IEnumVARIANT::Reset", COMError(hr))
	}
	return nil
}

type FirewallRuleCollection struct {
	object *wrappers.INetFwRules
}

func (self *FirewallRuleCollection) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *FirewallRuleCollection) GetCount() (int, error) {
	if self.object == nil {
		return 0, NewWindowsError("INetFwRules::get_Count", COMErrorPointer)
	}
	var count int32
	if hr := self.object.Get_Count(&count); wrappers.FAILED(hr) {
		return 0, NewWindowsError("INetFwRules::get_Count", COMError(hr))
	}
	return int(count), nil
}

func (self *FirewallRuleCollection) Add(rule *FirewallRule) error {
	if self.object == nil {
		return NewWindowsError("INetFwRules::Add", COMErrorPointer)
	}
	if hr := self.object.Add(rule.object); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRules::Add", COMError(hr))
	}
	return nil
}

func (self *FirewallRuleCollection) Remove(name string) error {
	if self.object == nil {
		return NewWindowsError("INetFwRules::Remove", COMErrorPointer)
	}
	nameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(name))
	defer wrappers.SysFreeString(nameRaw)
	if hr := self.object.Remove(nameRaw); wrappers.FAILED(hr) {
		return NewWindowsError("INetFwRules::Remove", COMError(hr))
	}
	return nil
}

func (self *FirewallRuleCollection) Item(name string) (*FirewallRule, error) {
	if self.object == nil {
		return nil, NewWindowsError("INetFwRules::Item", COMErrorPointer)
	}
	nameRaw := wrappers.SysAllocString(syscall.StringToUTF16Ptr(name))
	defer wrappers.SysFreeString(nameRaw)
	var rule *wrappers.INetFwRule
	if hr := self.object.Item(nameRaw, &rule); wrappers.FAILED(hr) {
		return nil, NewWindowsError("INetFwRules::Item", COMError(hr))
	}
	return &FirewallRule{object: rule}, nil
}

func (self *FirewallRuleCollection) NewEnumerator() (*FirewallRuleEnumerator, error) {
	if self.object == nil {
		return nil, NewWindowsError("INetFwRules::get__NewEnum", COMErrorPointer)
	}
	var punkEnumerator *wrappers.IUnknown
	if hr := self.object.Get__NewEnum(&punkEnumerator); wrappers.FAILED(hr) {
		return nil, NewWindowsError("INetFwRules::get__NewEnum", COMError(hr))
	}
	defer punkEnumerator.Release()
	var enumerator uintptr
	if hr := punkEnumerator.QueryInterface(&wrappers.IID_IEnumVARIANT, &enumerator); wrappers.FAILED(hr) {
		return nil, NewWindowsError("IUnknown::QueryInterface", COMError(hr))
	}
	return &FirewallRuleEnumerator{object: (*wrappers.IEnumVARIANT)(unsafe.Pointer(enumerator))}, nil
}

type FirewallPolicy struct {
	object *wrappers.INetFwPolicy2
}

func NewFirewallPolicy() (*FirewallPolicy, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_NetFwPolicy2,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_INetFwPolicy2,
		&object)
	if wrappers.FAILED(hr) {
		return nil, NewWindowsError("CoCreateInstance", COMError(hr))
	}
	return &FirewallPolicy{object: (*wrappers.INetFwPolicy2)(unsafe.Pointer(object))}, nil
}

func (self *FirewallPolicy) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *FirewallPolicy) GetRules() (*FirewallRuleCollection, error) {
	if self.object == nil {
		return nil, NewWindowsError("INetFwPolicy2::get_Rules", COMErrorPointer)
	}
	var rules *wrappers.INetFwRules
	if hr := self.object.Get_Rules(&rules); wrappers.FAILED(hr) {
		return nil, NewWindowsError("INetFwPolicy2::get_Rules", COMError(hr))
	}
	return &FirewallRuleCollection{object: rules}, nil
}

type FirewallManager struct {
	object *wrappers.INetFwMgr
}

func NewFirewallManager() (*FirewallManager, error) {
	var object uintptr
	hr := wrappers.CoCreateInstance(
		&wrappers.CLSID_NetFwMgr,
		nil,
		wrappers.CLSCTX_INPROC_SERVER,
		&wrappers.IID_INetFwMgr,
		&object)
	if wrappers.FAILED(hr) {
		return nil, NewWindowsError("CoCreateInstance", COMError(hr))
	}
	return &FirewallManager{object: (*wrappers.INetFwMgr)(unsafe.Pointer(object))}, nil
}

func (self *FirewallManager) Close() error {
	if self.object != nil {
		self.object.Release()
		self.object = nil
	}
	return nil
}

func (self *FirewallManager) IsPortAllowed(imageFileName string, ipVersion FirewallIPVersion, portNumber int, localAddress string, ipProtocol FirewallProtocol) (allowed bool, restricted bool, err error) {
	if self.object == nil {
		err = COMErrorPointer
		return
	}
	var imageFileNameRaw *uint16
	if imageFileName != "" {
		imageFileNameRaw = wrappers.SysAllocString(syscall.StringToUTF16Ptr(imageFileName))
		defer wrappers.SysFreeString(imageFileNameRaw)
	}
	var localAddressRaw *uint16
	if localAddress != "" {
		localAddressRaw = wrappers.SysAllocString(syscall.StringToUTF16Ptr(localAddress))
		defer wrappers.SysFreeString(localAddressRaw)
	}
	var allowedRaw wrappers.VARIANT
	wrappers.VariantInit(&allowedRaw)
	defer wrappers.VariantClear(&allowedRaw)
	var restrictedRaw wrappers.VARIANT
	wrappers.VariantInit(&restrictedRaw)
	defer wrappers.VariantClear(&restrictedRaw)
	hr := self.object.IsPortAllowed(
		imageFileNameRaw,
		int32(ipVersion),
		int32(portNumber),
		localAddressRaw,
		int32(ipProtocol),
		&allowedRaw,
		&restrictedRaw)
	if wrappers.SUCCEEDED(hr) {
		allowed = allowedRaw.Vt == wrappers.VT_BOOL && int16(allowedRaw.Val[0]) != wrappers.VARIANT_FALSE
		restricted = restrictedRaw.Vt == wrappers.VT_BOOL && int16(restrictedRaw.Val[0]) != wrappers.VARIANT_FALSE
	} else {
		err = NewWindowsError("INetFwMgr::IsPortAllowed", COMError(hr))
	}
	return
}
