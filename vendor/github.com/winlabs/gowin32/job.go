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

type JobLimitFlags uint32

const (
	JobLimitWorkingSet              JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_WORKINGSET
	JobLimitProcessTime             JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_PROCESS_TIME
	JobLimitJobTime                 JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_JOB_TIME
	JobLimitActiveProcess           JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_ACTIVE_PROCESS
	JobLimitAffinity                JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_AFFINITY
	JobLimitPriorityClass           JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_PRIORITY_CLASS
	JobLimitPreserveJobTime         JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_PRESERVE_JOB_TIME
	JobLimitSchedulingClass         JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_SCHEDULING_CLASS
	JobLimitProcessMemory           JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_PROCESS_MEMORY
	JobLimitDieOnUnhandledException JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_DIE_ON_UNHANDLED_EXCEPTION
	JobLimitBreakawayOK             JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_BREAKAWAY_OK
	JobLimitSilentBreakawayOK       JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK
	JobLimitKillOnJobClose          JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	JobLimitSubsetAffinity          JobLimitFlags = wrappers.JOB_OBJECT_LIMIT_SUBSET_AFFINITY
)

type JobBasicLimitInfo struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              JobLimitFlags
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint
	Affinity                uintptr
	PriorityClass           uint
	SchedulingClass         uint
}

type JobExtendedLimitInfo struct {
	JobBasicLimitInfo
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

type JobUILimitFlags uint32

const (
	JobUILimitHandles          JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_HANDLES
	JobUILimitReadClipboard    JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_READCLIPBOARD
	JobUILimitWriteClipboard   JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_WRITECLIPBOARD
	JobUILimitSystemParameters JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_SYSTEMPARAMETERS
	JobUILimitDisplaySettings  JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_DISPLAYSETTINGS
	JobUILimitGlobalAtoms      JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_GLOBALATOMS
	JobUILimitDesktop          JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_DESKTOP
	JobUILimitExitWindows      JobUILimitFlags = wrappers.JOB_OBJECT_UILIMIT_EXITWINDOWS
)

type Job struct {
	handle syscall.Handle
}

func NewJob(name string) (*Job, error) {
	var nameRaw *uint16
	if name != "" {
		nameRaw = syscall.StringToUTF16Ptr(name)
	}
	hJob, err := wrappers.CreateJobObject(nil, nameRaw)
	if err != nil {
		return nil, NewWindowsError("CreateJobObject", err)
	}
	return &Job{handle: hJob}, nil
}

func OpenJob(name string) (*Job, error) {
	hJob, err := wrappers.OpenJobObject(
		wrappers.JOB_OBJECT_ALL_ACCESS,
		false,
		syscall.StringToUTF16Ptr(name))
	if err != nil {
		return nil, NewWindowsError("OpenJobObject", err)
	}
	return &Job{handle: hJob}, nil
}

func (self *Job) Close() error {
	if self.handle != 0 {
		if err := wrappers.CloseHandle(self.handle); err != nil {
			return NewWindowsError("CloseHandle", err)
		}
		self.handle = 0
	}
	return nil
}

func (self *Job) AssignProcess(pid uint) error {
	hProcess, err := wrappers.OpenProcess(
		wrappers.PROCESS_SET_QUOTA | wrappers.PROCESS_TERMINATE,
		false,
		uint32(pid))
	if err != nil {
		return NewWindowsError("OpenProcess", err)
	}
	defer wrappers.CloseHandle(hProcess)
	if err := wrappers.AssignProcessToJobObject(self.handle, hProcess); err != nil {
		return NewWindowsError("AssignProcessToJobObject", err)
	}
	return nil
}

func (self *Job) GetBasicLimitInfo() (*JobBasicLimitInfo, error) {
	var info wrappers.JOBOBJECT_BASIC_LIMIT_INFORMATION
	err := wrappers.QueryInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicLimitInformation,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil)
	if err != nil {
		return nil, NewWindowsError("QueryInformationJobObject", err)
	}
	return &JobBasicLimitInfo{
		PerProcessUserTimeLimit: info.PerProcessUserTimeLimit,
		PerJobUserTimeLimit:     info.PerJobUserTimeLimit,
		LimitFlags:              JobLimitFlags(info.LimitFlags),
		MinimumWorkingSetSize:   info.MinimumWorkingSetSize,
		MaximumWorkingSetSize:   info.MaximumWorkingSetSize,
		ActiveProcessLimit:      uint(info.ActiveProcessLimit),
		Affinity:                info.Affinity,
		PriorityClass:           uint(info.PriorityClass),
		SchedulingClass:         uint(info.SchedulingClass),
	}, nil
}

func (self *Job) GetExtendedLimitInfo() (*JobExtendedLimitInfo, error) {
	var info wrappers.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	err := wrappers.QueryInformationJobObject(
		self.handle,
		wrappers.JobObjectExtendedLimitInformation,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil)
	if err != nil {
		return nil, NewWindowsError("QueryInformationJobObject", err)
	}
	return &JobExtendedLimitInfo{
		JobBasicLimitInfo: JobBasicLimitInfo{
			PerProcessUserTimeLimit: info.BasicLimitInformation.PerProcessUserTimeLimit,
			PerJobUserTimeLimit:     info.BasicLimitInformation.PerJobUserTimeLimit,
			LimitFlags:              JobLimitFlags(info.BasicLimitInformation.LimitFlags),
			MinimumWorkingSetSize:   info.BasicLimitInformation.MinimumWorkingSetSize,
			MaximumWorkingSetSize:   info.BasicLimitInformation.MaximumWorkingSetSize,
			ActiveProcessLimit:      uint(info.BasicLimitInformation.ActiveProcessLimit),
			Affinity:                info.BasicLimitInformation.Affinity,
			PriorityClass:           uint(info.BasicLimitInformation.PriorityClass),
			SchedulingClass:         uint(info.BasicLimitInformation.SchedulingClass),
		},
		ProcessMemoryLimit:      info.ProcessMemoryLimit,
		JobMemoryLimit:          info.JobMemoryLimit,
		PeakProcessMemoryUsed:   info.PeakProcessMemoryUsed,
		PeakJobMemoryUsed:       info.PeakJobMemoryUsed,
	}, nil
}

func (self *Job) GetBasicUIRestrictions() (JobUILimitFlags, error) {
	var info wrappers.JOBOBJECT_BASIC_UI_RESTRICTIONS
	err := wrappers.QueryInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicUIRestrictions,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil)
	if err != nil {
		return 0, NewWindowsError("QueryInformationJobObject", err)
	}
	return JobUILimitFlags(info.UIRestrictionsClass), nil
}

func (self *Job) GetProcesses() ([]uint, error) {
	var info wrappers.JOBOBJECT_BASIC_PROCESS_ID_LIST
	err := wrappers.QueryInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicProcessIdList,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil)
	if err != nil && err != wrappers.ERROR_MORE_DATA {
		return nil, NewWindowsError("QueryInformationJobObject", err)
	}
	buf := make([]byte, unsafe.Sizeof(info) + unsafe.Sizeof(info.ProcessIdList[0])*uintptr(info.NumberOfAssignedProcesses - 1))
	err = wrappers.QueryInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicProcessIdList,
		&buf[0],
		uint32(len(buf)),
		nil)
	if err != nil {
		return nil, NewWindowsError("QueryInformationJobObject", err)
	}
	bufInfo := (*wrappers.JOBOBJECT_BASIC_PROCESS_ID_LIST)(unsafe.Pointer(&buf[0]))
	rawPids := make([]uintptr, bufInfo.NumberOfProcessIdsInList)
	wrappers.RtlMoveMemory(
		(*byte)(unsafe.Pointer(&rawPids[0])),
		(*byte)(unsafe.Pointer(&bufInfo.ProcessIdList[0])),
		uintptr(bufInfo.NumberOfProcessIdsInList)*unsafe.Sizeof(rawPids[0]))
	pids := make([]uint, bufInfo.NumberOfProcessIdsInList)
	for i, rawPid := range rawPids {
		pids[i] = uint(rawPid)
	}
	return pids, nil
}

func (self *Job) SetBasicLimitInfo(info *JobBasicLimitInfo) error {
	rawInfo := wrappers.JOBOBJECT_BASIC_LIMIT_INFORMATION{
		PerProcessUserTimeLimit: info.PerProcessUserTimeLimit,
		PerJobUserTimeLimit:     info.PerJobUserTimeLimit,
		LimitFlags:              uint32(info.LimitFlags),
		MinimumWorkingSetSize:   info.MinimumWorkingSetSize,
		MaximumWorkingSetSize:   info.MaximumWorkingSetSize,
		ActiveProcessLimit:      uint32(info.ActiveProcessLimit),
		Affinity:                info.Affinity,
		PriorityClass:           uint32(info.PriorityClass),
		SchedulingClass:         uint32(info.SchedulingClass),
	}
	err := wrappers.SetInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicLimitInformation,
		(*byte)(unsafe.Pointer(&rawInfo)),
		uint32(unsafe.Sizeof(rawInfo)))
	if err != nil {
		return NewWindowsError("SetInformationJobObject", err)
	}
	return nil
}

func (self *Job) SetExtendedLimitInfo(info *JobExtendedLimitInfo) error {
	rawInfo := wrappers.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: wrappers.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			PerProcessUserTimeLimit: info.PerProcessUserTimeLimit,
			PerJobUserTimeLimit:     info.PerJobUserTimeLimit,
			LimitFlags:              uint32(info.LimitFlags),
			MinimumWorkingSetSize:   info.MinimumWorkingSetSize,
			MaximumWorkingSetSize:   info.MaximumWorkingSetSize,
			ActiveProcessLimit:      uint32(info.ActiveProcessLimit),
			Affinity:                info.Affinity,
			PriorityClass:           uint32(info.PriorityClass),
			SchedulingClass:         uint32(info.SchedulingClass),
		},
		ProcessMemoryLimit:    info.ProcessMemoryLimit,
		JobMemoryLimit:        info.JobMemoryLimit,
		PeakProcessMemoryUsed: info.PeakProcessMemoryUsed,
		PeakJobMemoryUsed:     info.PeakJobMemoryUsed,
	}
	err := wrappers.SetInformationJobObject(
		self.handle,
		wrappers.JobObjectExtendedLimitInformation,
		(*byte)(unsafe.Pointer(&rawInfo)),
		uint32(unsafe.Sizeof(rawInfo)))
	if err != nil {
		return NewWindowsError("SetInformationJobObject", err)
	}
	return nil
}

func (self *Job) SetBasicUIRestrictions(flags JobUILimitFlags) error {
	info := wrappers.JOBOBJECT_BASIC_UI_RESTRICTIONS{
		UIRestrictionsClass: uint32(flags),
	}
	err := wrappers.SetInformationJobObject(
		self.handle,
		wrappers.JobObjectBasicUIRestrictions,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)))
	if err != nil {
		return NewWindowsError("SetInformationJobObject", err)
	}
	return nil
}

func (self *Job) ProcessInJob(pid uint) (bool, error) {
	hProcess, err := wrappers.OpenProcess(wrappers.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false, NewWindowsError("OpenProcess", err)
	}
	defer wrappers.CloseHandle(hProcess)
	var result bool
	if err := wrappers.IsProcessInJob(hProcess, self.handle, &result); err != nil {
		return false, NewWindowsError("IsProcessInJob", err)
	}
	return result, nil
}

func (self *Job) Terminate(exitCode uint) error {
	if err := wrappers.TerminateJobObject(self.handle, uint32(exitCode)); err != nil {
		return NewWindowsError("TerminateJobObject", err)
	}
	return nil
}
