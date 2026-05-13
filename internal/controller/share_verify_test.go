package controller_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-file-server/internal/config"
	"go-file-server/internal/controller"
	"go-file-server/internal/middleware"
	"go-file-server/internal/model"
	"go-file-server/internal/repository"
	"go-file-server/internal/service"
	"go-file-server/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func setupShareVerifyRouter(t *testing.T) (*gin.Engine, *service.SharingService, repository.SharingRepository, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	cfg := config.AppConfig
	workDir := cfg.Server.FileRoot

	shareRepo := repository.NewSQLiteSharingRepo(db)
	sharingService := service.NewSharingService(shareRepo, workDir)

	middleware.ResetShareVerifyLimiter()

	router := gin.New()
	controller.SetupPublicShareRoutes(router, sharingService)

	return router, sharingService, shareRepo, db
}

// createShareDirect creates a share in the DB with a bcrypt-hashed PIN.
// Returns the share and the raw PIN.
func createShareDirect(t *testing.T, repo repository.SharingRepository, id, path, authority, rawPin string, blocked bool, expiresAt time.Time) (*model.SharingInfo, string) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(rawPin), bcrypt.DefaultCost)
	require.NoError(t, err)

	share := &model.SharingInfo{
		ID:          id,
		Path:        path,
		PinHash:     string(hash),
		ExpiresAt:   expiresAt,
		Blocked:     blocked,
		Authority:   authority,
		Username:    "testuser",
		Description: "test share",
		CreatedAt:   time.Now(),
	}
	err = repo.Create(share)
	require.NoError(t, err)
	return share, rawPin
}

// ---------------------------------------------------------------------------
// Verify Share PIN
// ---------------------------------------------------------------------------

func TestVerifySharePIN_Success(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-verify-ok", "/shared/docs", "view", "123456", false, time.Now().Add(24*time.Hour))

	body := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Verification successful", resp["message"])
	assert.Equal(t, "view", resp["authority"])

	// should have shareJwt cookie set
	hasCookie := false
	cookieName := "shareJwt_" + share.ID
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			assert.NotEmpty(t, c.Value)
			hasCookie = true
		}
	}
	assert.True(t, hasCookie, "shareJwt cookie should be set")
}

func TestVerifySharePIN_Invalid(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, _ := createShareDirect(t, shareRepo, "share-wrong-pin", "/shared/docs", "view", "999999", false, time.Now().Add(24*time.Hour))

	body := map[string]string{
		"id":  share.ID,
		"pin": "000000",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "Invalid PIN")
}

func TestVerifySharePIN_Blocked(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-blocked", "/shared/docs", "view", "123456", true, time.Now().Add(24*time.Hour))

	body := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "blocked")
}

func TestVerifySharePIN_Expired(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-expired", "/shared/docs", "view", "123456", false, time.Now().Add(-1*time.Hour))

	body := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	assert.Equal(t, http.StatusGone, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "expired")
}

func TestVerifySharePIN_NotFound(t *testing.T) {
	router, _, _, _ := setupShareVerifyRouter(t)

	body := map[string]string{
		"id":  "nonexistent-id",
		"pin": "123456",
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

func TestVerifySharePIN_RateLimit(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-ratelimit", "/shared/docs", "view", "123456", false, time.Now().Add(24*time.Hour))

	body := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}

	// burst allows 5; 6th should be rate-limited
	for i := 0; i < 5; i++ {
		rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestVerifySharePIN_WrongPIN_RateLimit(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, _ := createShareDirect(t, shareRepo, "share-wrong-ratelimit", "/shared/docs", "view", "999999", false, time.Now().Add(24*time.Hour))

	wrongBody := map[string]string{
		"id":  share.ID,
		"pin": "000000",
	}

	for i := 0; i < 5; i++ {
		rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", wrongBody, nil)
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d should not be rate-limited", i+1)
		require.Equal(t, http.StatusUnauthorized, rec.Code, "request %d should be unauthorized", i+1)
	}

	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", wrongBody, nil)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

// ---------------------------------------------------------------------------
// Check Share Permission
// ---------------------------------------------------------------------------

func TestCheckSharePermission_ValidToken(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-check-ok", "/shared/docs", "view", "123456", false, time.Now().Add(24*time.Hour))

	// verify PIN to get shareJwt cookie
	verifyBody := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}
	verifyRec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", verifyBody, nil)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	cookieName := "shareJwt_" + share.ID
	var shareCookie *http.Cookie
	for _, c := range verifyRec.Result().Cookies() {
		if c.Name == cookieName {
			shareCookie = c
			break
		}
	}
	require.NotNil(t, shareCookie)

	// now check permission
	checkReq := httptest.NewRequest(http.MethodGet, "/api/share/check-permission/"+share.ID, nil)
	checkReq.AddCookie(shareCookie)
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)

	assert.Equal(t, http.StatusOK, checkRec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(checkRec.Body.Bytes(), &resp))
	assert.Equal(t, "Permission verified", resp["message"])
}

func TestCheckSharePermission_NoToken(t *testing.T) {
	router, _, _, _ := setupShareVerifyRouter(t)

	rec := testutil.MakeAuthRequest(t, router, http.MethodGet, "/api/share/check-permission/some-id", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "missing share token")
}

func TestCheckSharePermission_WrongShareID(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	shareA, pinA := createShareDirect(t, shareRepo, "share-check-a", "/shared/a", "view", "123456", false, time.Now().Add(24*time.Hour))
	shareB, _ := createShareDirect(t, shareRepo, "share-check-b", "/shared/b", "view", "654321", false, time.Now().Add(24*time.Hour))

	// verify share A to get its cookie
	verifyBody := map[string]string{
		"id":  shareA.ID,
		"pin": pinA,
	}
	verifyRec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", verifyBody, nil)
	require.Equal(t, http.StatusOK, verifyRec.Code)

	var shareCookie *http.Cookie
	for _, c := range verifyRec.Result().Cookies() {
		if c.Name == "shareJwt_"+shareA.ID {
			shareCookie = c
			break
		}
	}
	require.NotNil(t, shareCookie)

	// try to check permission for share B using share A's token (wrong cookie name will be looked up)
	checkReq := httptest.NewRequest(http.MethodGet, "/api/share/check-permission/"+shareB.ID, nil)
	checkReq.AddCookie(shareCookie)
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)

	// The cookie name is shareJwt_shareB.ID but the cookie we have is for shareA
	// So it won't find the cookie for shareB → "missing share token"
	assert.Equal(t, http.StatusUnauthorized, checkRec.Code)
}

// ---------------------------------------------------------------------------
// Never-Expiring Share Verify
// ---------------------------------------------------------------------------

func TestVerifySharePIN_NeverExpires(t *testing.T) {
	router, _, shareRepo, _ := setupShareVerifyRouter(t)

	share, pin := createShareDirect(t, shareRepo, "share-never-exp", "/shared/forever", "modify", "888888", false, model.NeverExpires)

	body := map[string]string{
		"id":  share.ID,
		"pin": pin,
	}
	rec := testutil.MakeAuthRequestJSON(t, router, http.MethodPost, "/api/share/verify", body, nil)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "modify", resp["authority"])
}
