//go:build unix

package util

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCommand_KillsProcessGroup verifies that cancelling the context kills
// not just the direct child but any process it spawned. This is what stops
// ffmpeg process trees from being orphaned and accumulating.
func TestNewCommand_KillsProcessGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process test in short mode")
	}

	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")

	// The parent shell spawns a long-lived child, records its PID, then waits.
	// Killing the process group must take the child down too.
	script := `sleep 300 & echo $! > "` + pidFile + `"; wait`
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := NewCommand(ctx, "bash", "-c", script)
	require.NoError(t, cmd.Start())

	var childPID int
	require.Eventually(t, func() bool {
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			return false
		}
		childPID = pid
		return true
	}, 5*time.Second, 50*time.Millisecond, "child PID should be written")

	cancel()
	require.Error(t, cmd.Wait(), "cancelled command should return an error")

	require.Eventually(t, func() bool {
		return syscall.Kill(childPID, 0) == syscall.ESRCH
	}, 5*time.Second, 50*time.Millisecond, "child process %d should have been killed", childPID)
}

// TestNewCommand_WaitIsBoundedWhenGrandchildHoldsPipe verifies the WaitDelay
// guard: when a detached grandchild keeps the stderr pipe open, Wait must still
// return instead of blocking forever (which would leak a goroutine and the
// caller's worker slot).
func TestNewCommand_WaitIsBoundedWhenGrandchildHoldsPipe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping pipe-leak test in short mode")
	}

	// The parent exits immediately but leaves a background child holding the
	// stderr pipe open for far longer than CommandWaitDelay.
	cmd := NewCommand(context.Background(), "sh", "-c", "sleep 120 & exit 0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start())
	pgid := cmd.Process.Pid
	// The group outlives the exited parent (the background sleep is a member);
	// reap it so the test does not leave a stray process behind.
	defer func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) }()

	start := time.Now()
	err := cmd.Wait()
	elapsed := time.Since(start)

	// A successful process exit with the pipe still open surfaces as
	// ErrWaitDelay once the bound expires — that is the guard working.
	assert.True(t, errors.Is(err, exec.ErrWaitDelay), "Wait should return ErrWaitDelay when I/O is held open, got %v", err)
	assert.GreaterOrEqual(t, elapsed, CommandWaitDelay-time.Second, "Wait should honour WaitDelay")
	assert.Less(t, elapsed, CommandWaitDelay+5*time.Second, "Wait must be bounded by WaitDelay")
}
