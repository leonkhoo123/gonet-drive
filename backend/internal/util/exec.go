package util

import (
	"context"
	"os/exec"
	"time"
)

// CommandWaitDelay bounds how long Cmd.Wait may block after the process has
// exited (or the context was cancelled). Without it, Wait also waits for the
// stdout/stderr pipes to reach EOF, which never happens if ffmpeg leaves a
// grandchild holding the write end — that leaks a goroutine and the slot in
// the caller's worker/semaphore forever. 10s is plenty for ffmpeg to flush.
const CommandWaitDelay = 10 * time.Second

// NewCommand builds an exec.Cmd for external media tools (ffmpeg/ffprobe and
// the prlimit wrapper) with leak-proof defaults:
//
//   - the command is tied to ctx and killed when ctx is done;
//   - the whole process group is killed (see killProcessGroup), so an ffmpeg
//     process tree cannot be orphaned and accumulate;
//   - Wait is always bounded by CommandWaitDelay.
//
// Use this instead of exec.CommandContext for every ffmpeg/ffprobe call so a
// single misbehaving process cannot leak a process, goroutine, or handler.
func NewCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error { return killProcessGroup(cmd) }
	cmd.WaitDelay = CommandWaitDelay
	configureProcessGroup(cmd)
	return cmd
}
