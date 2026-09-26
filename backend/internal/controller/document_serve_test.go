package controller_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeDocument_HTMLServedAsInertPlainText(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "evil.html",
		"<html><body><script>alert(document.domain)</script></body></html>")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/document/read/file/evil.html", nil, accessCookie)

	require.Equal(t, http.StatusOK, rec.Code)
	// Must never be served as text/html, or the browser would execute the script
	// in the application's origin (stored XSS).
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, rec.Body.String(), "alert(document.domain)")
}

func TestServeDocument_PdfKeepsPdfContentType(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	writeTestFile(t, cfg.Server.FileRoot, "doc.pdf", "%PDF-1.4\n%%EOF")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/document/read/file/doc.pdf", nil, accessCookie)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
}

func TestServeDocument_SymlinkEscapeForbidden(t *testing.T) {
	router, cfg, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("top secret"), 0644))
	require.NoError(t, os.Symlink(secret, filepath.Join(cfg.Server.FileRoot, "escape.txt")))

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/document/read/file/escape.txt", nil, accessCookie)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "top secret")
}

func TestServeDocument_TraversalRejected(t *testing.T) {
	router, _, db := setupFileRouter(t)
	testutil.CreateTestUser(t, db, "fileuser", "pass123", "user")
	accessCookie := testutil.LoginAndGetCookie(t, router, "fileuser", "pass123")

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet,
		"/api/user/document/read/file/../../etc/passwd", nil, accessCookie)

	// Gin may normalize the path away (404) or hand it to the handler which
	// rejects it (403); either way it must never be served.
	assert.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, rec.Code)
}
