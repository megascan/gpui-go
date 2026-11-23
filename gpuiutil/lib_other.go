//go:build !windows

package gpuiutil

import "os/exec"

// NewStdCommand returns a command for program.
func NewStdCommand(program string, args ...string) *exec.Cmd {
	return exec.Command(program, args...)
}
