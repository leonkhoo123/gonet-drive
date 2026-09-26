package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewCommand_SetsLeakGuards guards against a refactor dropping the
// WaitDelay/Cancel that keep ffmpeg from leaking processes and goroutines.
func TestNewCommand_SetsLeakGuards(t *testing.T) {
	cmd := NewCommand(context.Background(), "true")
	assert.Equal(t, CommandWaitDelay, cmd.WaitDelay, "WaitDelay must bound Cmd.Wait")
	assert.NotNil(t, cmd.Cancel, "Cancel must kill the process on context cancellation")
}
