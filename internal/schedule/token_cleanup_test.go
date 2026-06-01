package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUntilNext_BeforeTarget(t *testing.T) {
	now := time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)
	d := untilNext(now, 3, 0)
	assert.Equal(t, 2*time.Hour, d)
}

func TestUntilNext_AfterTarget(t *testing.T) {
	now := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	d := untilNext(now, 3, 0)
	assert.Equal(t, 22*time.Hour, d)
}

func TestUntilNext_ExactlyTarget(t *testing.T) {
	now := time.Date(2026, 6, 1, 3, 0, 0, 0, time.UTC)
	d := untilNext(now, 3, 0)
	assert.Equal(t, time.Duration(0), d)
}
