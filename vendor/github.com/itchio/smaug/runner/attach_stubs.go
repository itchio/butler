// +build !windows

package runner

func getAttachRunner(params *RunnerParams) (Runner, error) {
	return nil, nil
}
