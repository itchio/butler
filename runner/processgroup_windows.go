// +build windows

package runner

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/itchio/butler/runner/execas"
	"github.com/itchio/butler/runner/syscallex"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type processGroup struct {
	consumer  *state.Consumer
	cmd       *execas.Cmd
	ctx       context.Context
	jobObject syscall.Handle
}

func NewProcessGroup(consumer *state.Consumer, cmd *execas.Cmd, ctx context.Context) (*processGroup, error) {
	pg := &processGroup{
		consumer:  consumer,
		cmd:       cmd,
		ctx:       ctx,
		jobObject: syscall.InvalidHandle,
	}
	return pg, nil
}

func (pg *processGroup) AfterStart() error {
	var err error
	pg.jobObject, err = syscallex.CreateJobObject(nil, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	jobObjectInfo := new(syscallex.JobObjectExtendedLimitInformation)
	jobObjectInfo.BasicLimitInformation.LimitFlags = syscallex.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	jobObjectInfoPtr := uintptr(unsafe.Pointer(jobObjectInfo))
	jobObjectInfoSize := unsafe.Sizeof(*jobObjectInfo)

	err = syscallex.SetInformationJobObject(
		pg.jobObject,
		syscallex.JobObjectInfoClass_JobObjectExtendedLimitInformation,
		jobObjectInfoPtr,
		jobObjectInfoSize,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	processHandle := pg.cmd.SysProcAttr.ProcessHandle

	pg.consumer.Infof("process handle: %x", uintptr(processHandle))
	err = syscallex.AssignProcessToJobObject(pg.jobObject, processHandle)
	if err != nil {
		pg.consumer.Warnf("No job object support (%s)", err.Error())
		pg.consumer.Warnf("The 'Running...' indicator and 'Force close' functionality will not work as expected, and ")
		syscall.CloseHandle(pg.jobObject)
		pg.jobObject = syscall.InvalidHandle
		return nil
	}

	return nil
}

func (pg *processGroup) Wait() error {
	waitDone := make(chan error)
	go func() {
		if pg.jobObject == syscall.InvalidHandle {
			pg.consumer.Infof("Waiting on single process...")
			waitDone <- pg.cmd.Wait()
		} else {
			pg.consumer.Infof("Waiting on whole job object...")
			_, err := syscall.WaitForSingleObject(pg.jobObject, syscall.INFINITE)
			waitDone <- err
		}
	}()

	select {
	case <-pg.ctx.Done():
		if pg.jobObject == syscall.InvalidHandle {
			pid := uint32(pg.cmd.Process.Pid)
			pg.consumer.Infof("Killing single process %d", pid)
			pg.cmd.Process.Kill()
		} else {
			pg.consumer.Infof("Attempting to kill entire job object...")
			var processIdList syscallex.JobObjectBasicProcessIdList
			processIdListPtr := uintptr(unsafe.Pointer(&processIdList))
			processIdListSize := unsafe.Sizeof(processIdList)

			pg.consumer.Infof("Querying job object...")
			err := syscallex.QueryInformationJobObject(
				pg.jobObject,
				syscallex.JobObjectInfoClass_JobObjectBasicProcessIdList,
				processIdListPtr,
				processIdListSize,
				0,
			)
			if err != nil {
				pg.consumer.Infof("Querying job object error (!)")
				ignoreError := false
				if en, ok := err.(syscall.Errno); ok {
					if en == syscall.ERROR_MORE_DATA {
						// that's expected, the struct we pass has only room for 1 process
						ignoreError = true
					}
				}

				if !ignoreError {
					return errors.WithStack(err)
				}
			}

			pg.consumer.Infof("%d processes still in job object", processIdList.NumberOfAssignedProcesses)
			pg.consumer.Infof("%d processes in our list", processIdList.NumberOfProcessIdsInList)
			for i := uint32(0); i < processIdList.NumberOfProcessIdsInList; i++ {
				pid := uint32(processIdList.ProcessIdList[i])
				pg.consumer.Infof("- PID %d", pid)
				err := terminateProcess(pid, 0)
				if err != nil {
					pg.consumer.Warnf("Could not kill pid %d: %s", pid, err.Error())
				}
			}
		}
	case err := <-waitDone:
		pg.consumer.Infof("Wait done")
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func terminateProcess(pid uint32, exitcode int) error {
	h, err := syscall.OpenProcess(syscall.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return errors.WithStack(err)
	}
	defer syscall.CloseHandle(h)
	err = syscall.TerminateProcess(h, uint32(exitcode))
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
