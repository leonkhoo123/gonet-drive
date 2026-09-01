package ws

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	authgin "github.com/leonkhoo123/gonet-auth/adapters/gin"
	"github.com/stretchr/testify/assert"
)

func TestResolveUsername(t *testing.T) {
	tests := []struct {
		name      string
		authKey   string
		shareID   string
		legacyKey string
		want      string
	}{
		{
			name:    "authenticated user takes priority",
			authKey: "alice",
			shareID: "abc",
			want:    "alice",
		},
		{
			name:    "share connection uses share prefix",
			shareID: "abc",
			want:    "share:abc",
		},
		{
			name:      "falls back to legacy username key",
			legacyKey: "legacyuser",
			want:      "legacyuser",
		},
		{
			name: "empty when no identity present",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			if tt.authKey != "" {
				c.Set(authgin.KeyUsername, tt.authKey)
			}
			if tt.shareID != "" {
				c.Set("share_id", tt.shareID)
			}
			if tt.legacyKey != "" {
				c.Set("username", tt.legacyKey)
			}
			assert.Equal(t, tt.want, resolveUsername(c))
		})
	}
}
