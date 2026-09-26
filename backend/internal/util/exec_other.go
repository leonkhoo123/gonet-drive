//go:build !unix

package util

import "os/exec"

// configureProcessGroup is a no-op on platforms without POSIX process groups.
func configureProcessGroup(_ *exec.Cmd) {}

// killProcessGroup falls back to killing the direct child only.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
