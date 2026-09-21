//go:build !windows

package native

import (
	"fmt"
	"os/exec"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpretRunErrorSeesTheSignalThroughWrapping(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	require.NoError(t, cmd.Start())
	require.NoError(t, cmd.Process.Kill())
	err := cmd.Wait()
	require.Error(t, err)

	exitCode, signal, err := interpretRunError(fmt.Errorf("%w", err))
	require.NoError(t, err)
	assert.Equal(t, -1, exitCode)
	assert.Equal(t, syscall.SIGKILL, signal)
}

func TestInterpretRunErrorExitCode(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 3")
	err := cmd.Run()
	require.Error(t, err)

	exitCode, signal, err := interpretRunError(err)
	require.NoError(t, err)
	assert.Equal(t, 3, exitCode)
	assert.Equal(t, syscall.Signal(0), signal)
}
