// +build windows

package runner

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/launch/launchers/native/runner/syscallex"
	"github.com/itchio/wharf/state"
)

var setupDone = false
var butlerJobObject syscall.Handle

func SetupJobObject(consumer *state.Consumer) error {
	if setupDone {
		return nil
	}

	setupDone = true

	jobObject, err := syscallex.CreateJobObject(nil, nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	jobObjectInfo := new(syscallex.JobObjectExtendedLimitInformation)
	jobObjectInfo.BasicLimitInformation.LimitFlags = syscallex.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	jobObjectInfoPtr := uintptr(unsafe.Pointer(jobObjectInfo))
	jobObjectInfoSize := unsafe.Sizeof(*jobObjectInfo)

	err = syscallex.SetInformationJobObject(
		jobObject,
		syscallex.JobObjectInfoClass_JobObjectExtendedLimitInformation,
		jobObjectInfoPtr,
		jobObjectInfoSize,
	)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	currentProcess, err := syscall.GetCurrentProcess()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = syscallex.AssignProcessToJobObject(jobObject, currentProcess)
	if err != nil {
		consumer.Warnf("No job object support (%s)", err.Error())
		consumer.Warnf("The 'Running...' indicator and 'Force close' functionality will not work as expected, and ")
		return nil
	}

	butlerJobObject = jobObject
	return nil
}

func WaitJobObject(consumer *state.Consumer) error {
	if butlerJobObject == 0 {
		return nil
	}

	var processIdList syscallex.JobObjectBasicProcessIdList
	processIdListPtr := uintptr(unsafe.Pointer(&processIdList))
	processIdListSize := unsafe.Sizeof(processIdList)

	var rounds int64

	for {
		err := syscallex.QueryInformationJobObject(
			butlerJobObject,
			syscallex.JobObjectInfoClass_JobObjectBasicProcessIdList,
			processIdListPtr,
			processIdListSize,
			0,
		)
		if err != nil {
			ignoreError := false
			if en, ok := err.(syscall.Errno); ok {
				if en == syscall.ERROR_MORE_DATA {
					// that's expected, the struct we pass has only room for 1 process
					ignoreError = true
				}
			}

			if !ignoreError {
				return errors.Wrap(err, 0)
			}
		}

		if processIdList.NumberOfAssignedProcesses <= 1 {
			// it's just us left? quit!
			consumer.Infof("Done waiting for job object after %d rounds", rounds)
			return nil
		}

		// don't busywait - 500ms is enough
		time.Sleep(500 * time.Millisecond)

		rounds++
	}
}
