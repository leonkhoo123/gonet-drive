//go:build unix

package util

import (
	"errors"
	"os/exec"
	"syscall"
)

// configureProcessGroup puts each external command in its own process group so
// the whole ffmpeg process tree can be signalled at once.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup SIGKILLs the command's entire process group. The negative
// pid targets the group created by Setpgid, so ffmpeg's descendants die with
// it instead of being orphaned. A missing group (ESRCH) is not an error: the
// process already exited, which is exactly what we want.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
