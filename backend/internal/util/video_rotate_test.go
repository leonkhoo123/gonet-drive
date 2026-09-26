package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjustRotationAngle(t *testing.T) {
	cases := []struct {
		name        string
		current     int
		rotateAngle int
		want        int
	}{
		{"no rotation", 0, 0, 0},
		{"quarter turn clockwise", 0, 90, -90},
		{"half turn", 0, 180, -180},
		{"three quarter turn", 0, 270, -270},
		{"cancels existing rotation", 90, 90, 0},
		{"accumulates and normalises", 270, 90, -180},
		{"negative delta wraps into range", 0, -90, -270},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, AdjustRotationAngle(tc.current, tc.rotateAngle))
		})
	}
}
