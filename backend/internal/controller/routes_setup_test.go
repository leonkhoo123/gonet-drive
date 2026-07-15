package controller_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go-file-server/internal/controller"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------- Finding 4.4: sanitizeDeviceInfo ----------

func TestSanitizeDeviceInfo_PlainString(t *testing.T) {
	input := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	result := controller.SanitizeDeviceInfo(input)
	assert.Equal(t, input, result)
}

func TestSanitizeDeviceInfo_EscapesHTML(t *testing.T) {
	input := `<script>alert("xss")</script>`
	result := controller.SanitizeDeviceInfo(input)
	assert.NotContains(t, result, "<script>")
	assert.Contains(t, result, "&lt;script&gt;")
}

func TestSanitizeDeviceInfo_EscapesAmpersand(t *testing.T) {
	input := "foo & bar"
	result := controller.SanitizeDeviceInfo(input)
	assert.Equal(t, "foo &amp; bar", result)
}

func TestSanitizeDeviceInfo_EscapesQuotes(t *testing.T) {
	input := `He said "hello" and 'bye'`
	result := controller.SanitizeDeviceInfo(input)
	assert.Contains(t, result, "&#34;")
	assert.Contains(t, result, "&#39;")
}

func TestSanitizeDeviceInfo_TruncatesTo512(t *testing.T) {
	long := make([]byte, 600)
	for i := range long {
		long[i] = 'A'
	}
	result := controller.SanitizeDeviceInfo(string(long))
	assert.Len(t, result, 512)
}

func TestSanitizeDeviceInfo_EmptyString(t *testing.T) {
	result := controller.SanitizeDeviceInfo("")
	assert.Equal(t, "", result)
}

// ---------- Finding 7.1: securityHeadersMiddleware ----------

func TestSecurityHeadersMiddleware_SetsHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(controller.SecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=")
	assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "", rec.Header().Get("Server"))
}
