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

type PrivilegeName string

const (
	PrivilegeCreateToken          PrivilegeName = wrappers.SE_CREATE_TOKEN_NAME
	PrivilegeAssignPrimaryToken   PrivilegeName = wrappers.SE_ASSIGNPRIMARYTOKEN_NAME
	PrivilegeLockMemory           PrivilegeName = wrappers.SE_LOCK_MEMORY_NAME
	PrivilegeIncreaseQuota        PrivilegeName = wrappers.SE_INCREASE_QUOTA_NAME
	PrivilegeUnsolicitedInput     PrivilegeName = wrappers.SE_UNSOLICITED_INPUT_NAME
	PrivilegeMachineAccount       PrivilegeName = wrappers.SE_MACHINE_ACCOUNT_NAME
	PrivilegeTCB                  PrivilegeName = wrappers.SE_TCB_NAME
	PrivilegeSecurity             PrivilegeName = wrappers.SE_SECURITY_NAME
	PrivilegeTakeOwnership        PrivilegeName = wrappers.SE_TAKE_OWNERSHIP_NAME
	PrivilegeLoadDriver           PrivilegeName = wrappers.SE_LOAD_DRIVER_NAME
	PrivilegeSystemProfile        PrivilegeName = wrappers.SE_SYSTEM_PROFILE_NAME
	PrivilegeSystemTime           PrivilegeName = wrappers.SE_SYSTEMTIME_NAME
	PrivilegeProfileSingleProcess PrivilegeName = wrappers.SE_PROF_SINGLE_PROCESS_NAME
	PrivilegeIncreaseBasePriority PrivilegeName = wrappers.SE_INC_BASE_PRIORITY_NAME
	PrivilegeCreatePagefile       PrivilegeName = wrappers.SE_CREATE_PAGEFILE_NAME
	PrivilegeCreatePermanent      PrivilegeName = wrappers.SE_CREATE_PERMANENT_NAME
	PrivilegeBackup               PrivilegeName = wrappers.SE_BACKUP_NAME
	PrivilegeRestore              PrivilegeName = wrappers.SE_RESTORE_NAME
	PrivilegeShutdown             PrivilegeName = wrappers.SE_SHUTDOWN_NAME
	PrivilegeDebug                PrivilegeName = wrappers.SE_DEBUG_NAME
	PrivilegeAudit                PrivilegeName = wrappers.SE_AUDIT_NAME
	PrivilegeSystemEnvironment    PrivilegeName = wrappers.SE_SYSTEM_ENVIRONMENT_NAME
	PrivilegeChangeNotify         PrivilegeName = wrappers.SE_CHANGE_NOTIFY_NAME
	PrivilegeRemoteShutdown       PrivilegeName = wrappers.SE_REMOTE_SHUTDOWN_NAME
	PrivilegeUndock               PrivilegeName = wrappers.SE_UNDOCK_NAME
	PrivilegeSyncAgent            PrivilegeName = wrappers.SE_SYNC_AGENT_NAME
	PrivilegeEnableDelegation     PrivilegeName = wrappers.SE_ENABLE_DELEGATION_NAME
	PrivilegeManageVolume         PrivilegeName = wrappers.SE_MANAGE_VOLUME_NAME
	PrivilegeImpersonate          PrivilegeName = wrappers.SE_IMPERSONATE_NAME
	PrivilegeCreateGlobal         PrivilegeName = wrappers.SE_CREATE_GLOBAL_NAME
	PrivilegeTrustedCredManAccess PrivilegeName = wrappers.SE_TRUSTED_CREDMAN_ACCESS_NAME
	PrivilegeRelabel              PrivilegeName = wrappers.SE_RELABEL_NAME
	PrivilegeIncreaseWorkingSet   PrivilegeName = wrappers.SE_INC_WORKING_SET_NAME
	PrivilegeTimeZone             PrivilegeName = wrappers.SE_TIME_ZONE_NAME
	PrivilegeCreateSymbolicLink   PrivilegeName = wrappers.SE_CREATE_SYMBOLIC_LINK_NAME
)

type Privilege struct {
	luid wrappers.LUID
}

func GetPrivilege(name PrivilegeName) (*Privilege, error) {
	var luid wrappers.LUID
	err := wrappers.LookupPrivilegeValue(
		nil,
		syscall.StringToUTF16Ptr(string(name)),
		&luid)
	if err != nil {
		return nil, NewWindowsError("LookupPrivilegeValue", err)
	}
	return &Privilege{luid: luid}, nil
}

type SecurityIDType int32

const (
	SecurityIDTypeUser           SecurityIDType = wrappers.SidTypeUser
	SecurityIDTypeGroup          SecurityIDType = wrappers.SidTypeGroup
	SecurityIDTypeDomain         SecurityIDType = wrappers.SidTypeDomain
	SecurityIDTypeAlias          SecurityIDType = wrappers.SidTypeAlias
	SecurityIDTypeWellKnownGroup SecurityIDType = wrappers.SidTypeWellKnownGroup
	SecurityIDTypeDeletedAccount SecurityIDType = wrappers.SidTypeDeletedAccount
	SecurityIDTypeInvalid        SecurityIDType = wrappers.SidTypeInvalid
	SecurityIDTypeUnknown        SecurityIDType = wrappers.SidTypeUnknown
	SecurityIDTypeComputer       SecurityIDType = wrappers.SidTypeComputer
	SecurityIDTypeLabel          SecurityIDType = wrappers.SidTypeLabel
)

type SecurityID struct {
	sid *wrappers.SID
}

func (self SecurityID) GetLength() uint {
	return uint(wrappers.GetLengthSid(self.sid))
}

func (self SecurityID) Copy() (SecurityID, error) {
	length := self.GetLength()
	buf := make([]byte, length)
	sid := (*wrappers.SID)(unsafe.Pointer(&buf[0]))
	err := wrappers.CopySid(uint32(length), sid, self.sid)
	if err != nil {
		return SecurityID{}, NewWindowsError("CopySid", err)
	}
	return SecurityID{sid}, nil
}

func (self SecurityID) Equal(other SecurityID) bool {
	return wrappers.EqualSid(self.sid, other.sid)
}

func (self SecurityID) String() (string, error) {
	var stringSid *uint16
	if err := wrappers.ConvertSidToStringSid(self.sid, &stringSid); err != nil {
		return "", NewWindowsError("ConvertSidToStringSid", err)
	}
	defer wrappers.LocalFree(syscall.Handle(unsafe.Pointer(stringSid)))
	return LpstrToString(stringSid), nil
}

func GetFileOwner(path string) (SecurityID, error) {
	var needed uint32
	wrappers.GetFileSecurity(
		syscall.StringToUTF16Ptr(path),
		wrappers.OWNER_SECURITY_INFORMATION,
		nil,
		0,
		&needed)
	buf := make([]byte, needed)
	err := wrappers.GetFileSecurity(
		syscall.StringToUTF16Ptr(path),
		wrappers.OWNER_SECURITY_INFORMATION,
		&buf[0],
		needed,
		&needed)
	if err != nil {
		return SecurityID{}, NewWindowsError("GetFileSecurity", err)
	}
	var ownerSid *wrappers.SID
	if err := wrappers.GetSecurityDescriptorOwner(&buf[0], &ownerSid, nil); err != nil {
		return SecurityID{}, NewWindowsError("GetSecurityDescriptorOwner", err)
	}
	return SecurityID{ownerSid}, nil
}

func GetLocalAccountByName(accountName string) (SecurityID, string, SecurityIDType, error) {
	var neededForSid uint32
	var neededForDomain uint32
	var use int32
	wrappers.LookupAccountName(
		nil,
		syscall.StringToUTF16Ptr(accountName),
		nil,
		&neededForSid,
		nil,
		&neededForDomain,
		&use)
	sidBuf := make([]byte, neededForSid)
	sid := (*wrappers.SID)(unsafe.Pointer(&sidBuf[0]))
	domainBuf := make([]uint16, neededForDomain)
	err := wrappers.LookupAccountName(
		nil,
		syscall.StringToUTF16Ptr(accountName),
		sid,
		&neededForSid,
		&domainBuf[0],
		&neededForDomain,
		&use)
	if err != nil {
		return SecurityID{}, "", 0, NewWindowsError("LookupAccountName", err)
	}
	return SecurityID{sid}, syscall.UTF16ToString(domainBuf), SecurityIDType(use), nil
}

func BeginImpersonateSelf() error {
	if err := wrappers.ImpersonateSelf(wrappers.SecurityImpersonation); err != nil {
		return NewWindowsError("ImpersonateSelf", nil)
	}
	return nil
}

func EndImpersonate() error {
	if err := wrappers.RevertToSelf(); err != nil {
		return NewWindowsError("RevertToSelf", nil)
	}
	return nil
}

type Token struct {
	handle syscall.Handle
}

func OpenCurrentProcessToken() (*Token, error) {
	hProcess := wrappers.GetCurrentProcess()
	var hToken syscall.Handle
	if err := wrappers.OpenProcessToken(hProcess, wrappers.TOKEN_QUERY, &hToken); err != nil {
		return nil, NewWindowsError("OpenProcessToken", err)
	}
	return &Token{handle: hToken}, nil
}

func OpenOtherProcessToken(pid uint) (*Token, error) {
	hProcess, err := wrappers.OpenProcess(wrappers.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil, NewWindowsError("OpenProcess", err)
	}
	defer syscall.CloseHandle(hProcess)
	var hToken syscall.Handle
	if err := wrappers.OpenProcessToken(hProcess, wrappers.TOKEN_QUERY, &hToken); err != nil {
		return nil, NewWindowsError("OpenProcessToken", err)
	}
	return &Token{handle: hToken}, nil
}

func OpenCurrentThreadToken(openAsSelf bool) (*Token, error) {
	hThread := wrappers.GetCurrentThread()
	var hToken syscall.Handle
	err := wrappers.OpenThreadToken(
		hThread,
		wrappers.TOKEN_QUERY|wrappers.TOKEN_ADJUST_PRIVILEGES,
		openAsSelf,
		&hToken)
	if err != nil {
		return nil, NewWindowsError("OpenThreadToken", err)
	}
	return &Token{handle: hToken}, nil
}

func (self *Token) Close() error {
	if self.handle != 0 {
		if err := wrappers.CloseHandle(self.handle); err != nil {
			return NewWindowsError("CloseHandle", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *Token) EnablePrivilege(privilege *Privilege, enable bool) error {
	tokenPrivileges := wrappers.TOKEN_PRIVILEGES{
		PrivilegeCount: 1,
		Privileges: [1]wrappers.LUID_AND_ATTRIBUTES{
			{Luid: privilege.luid},
		},
	}
	if enable {
		tokenPrivileges.Privileges[0].Attributes = wrappers.SE_PRIVILEGE_ENABLED
	}
	if err := wrappers.AdjustTokenPrivileges(self.handle, false, &tokenPrivileges, 0, nil, nil); err != nil {
		return NewWindowsError("AdjustTokenPrivileges", err)
	}
	return nil
}

func (self *Token) GetOwner() (SecurityID, error) {
	var needed uint32
	wrappers.GetTokenInformation(
		self.handle,
		wrappers.TokenOwner,
		nil,
		0,
		&needed)
	buf := make([]byte, needed)
	err := wrappers.GetTokenInformation(
		self.handle,
		wrappers.TokenOwner,
		&buf[0],
		needed,
		&needed)
	if err != nil {
		return SecurityID{}, NewWindowsError("GetTokenInformation", err)
	}
	ownerData := (*wrappers.TOKEN_OWNER)(unsafe.Pointer(&buf[0]))
	sid, err := SecurityID{ownerData.Owner}.Copy()
	if err != nil {
		return SecurityID{}, err
	}
	return sid, nil
}

func IsAdmin() (bool, error) {
	var sid *wrappers.SID
	err := wrappers.AllocateAndInitializeSid(
		&wrappers.SECURITY_NT_AUTHORITY,
		2,
		wrappers.SECURITY_BUILTIN_DOMAIN_RID,
		wrappers.DOMAIN_ALIAS_RID_ADMINS,
		0,
		0,
		0,
		0,
		0,
		0,
		&sid)
	if err != nil {
		return false, NewWindowsError("AllocateAndInitializeSid", err)
	}
	defer wrappers.FreeSid(sid)
	var isAdmin bool
	if err := wrappers.CheckTokenMembership(0, sid, &isAdmin); err != nil {
		return false, NewWindowsError("CheckTokenMembership", err)
	}
	return isAdmin, nil
}
