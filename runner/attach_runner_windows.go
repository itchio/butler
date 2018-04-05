// +build windows

package runner

import (
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/itchio/butler/runner/syscallex"
	"github.com/itchio/butler/runner/winutil"
	"github.com/pkg/errors"
)

func getAttachRunner(params *RunnerParams) (Runner, error) {
	consumer := params.RequestContext.Consumer

	snapshot, err := syscallex.CreateToolhelp32Snapshot(
		syscallex.TH32CS_SNAPPROCESS,
		0,
	)
	if err != nil {
		return nil, errors.WithMessage(err, "could not create toolhelp32 snapshot")
	}

	defer winutil.SafeRelease(uintptr(snapshot))

	var entry syscallex.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	err = syscallex.Process32First(snapshot, &entry)
	if err != nil {
		return nil, errors.WithMessage(err, "could not get first process")
	}

	basePath := filepath.Base(params.FullTargetPath)
	numInspected := 0
	for {
		matches := false

		err := func() error {
			process, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, entry.ProcessID)
			if err != nil {
				return errors.WithStack(err)
			}
			defer winutil.SafeRelease(uintptr(process))

			name, err := syscallex.QueryFullProcessImageName(process, 0)
			if err != nil {
				return errors.WithStack(err)
			}
			numInspected++

			runningPath := filepath.Clean(name)
			if runningPath == params.FullTargetPath {
				matches = true
			} else if filepath.Base(runningPath) == basePath {
				consumer.Infof("%s (PID %d) looks like us, but it's not us", runningPath, entry.ProcessID)
			}

			return nil
		}()
		if err != nil {
			consumer.Debugf("Could not get full image name for PID (%d): %s", entry.ProcessID, err.Error())
		}

		if matches {
			consumer.Infof("Found running copy of %s (PID %d)", params.FullTargetPath, entry.ProcessID)
			ar := &attachRunner{
				params: params,
				pid:    entry.ProcessID,
			}
			return ar, nil
		}

		err = syscallex.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}

	consumer.Infof("Inspected %d processes, could not find running copy of %s", numInspected, params.FullTargetPath)

	return nil, nil
}

type attachRunner struct {
	params *RunnerParams
	pid    uint32
}

var _ Runner = (*attachRunner)(nil)

func (ar *attachRunner) Prepare() error {
	return nil
}

func (ar *attachRunner) Run() error {
	consumer := ar.params.RequestContext.Consumer

	cancel := make(chan struct{})
	defer close(cancel)

	go func() {
		select {
		case <-cancel:
			// nothing
		case <-ar.params.Ctx.Done():
			err := terminateProcess(ar.pid, 1)
			if err != nil {
				consumer.Warnf("Could not terminate PID (%d): %v", err)
			}
		}
	}()

	processHandle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, ar.pid)
	if err != nil {
		return errors.WithStack(err)
	}
	defer winutil.SafeRelease(uintptr(processHandle))

	consumer.Infof("Attached to PID (%d)", ar.pid)
	_, err = syscall.WaitForSingleObject(processHandle, syscall.INFINITE)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
