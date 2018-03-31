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

	numInspected := 0
	for {
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
				consumer.Infof("Found running copy of %s (PID %d)", params.FullTargetPath, entry.ProcessID)
			}

			return nil
		}()
		if err != nil {
			consumer.Debugf("Could not get full image name for PID (%d): %s", entry.ProcessID, err.Error())
		}

		err = syscallex.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}

	return nil, nil
}
