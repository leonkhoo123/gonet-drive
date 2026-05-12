package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertJSONResponse checks status code and unmarshalled JSON body.
func AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedBody string) {
	t.Helper()
	assert.Equal(t, expectedStatus, recorder.Code)
	assert.JSONEq(t, expectedBody, recorder.Body.String())
}

// AssertAuthError ensures the response is a JSON error with proper auth error status.
func AssertAuthError(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()
	assert.Equal(t, expectedStatus, recorder.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &resp)
	require.NoError(t, err, "response body should be valid JSON")

	_, hasError := resp["error"]
	assert.True(t, hasError, "response should contain an 'error' field")
}

// AssertNoAuthCookie ensures the response has cleared (MaxAge=-1) the named cookie.
func AssertNoAuthCookie(t *testing.T, recorder *httptest.ResponseRecorder, cookieName string) {
	t.Helper()
	for _, c := range recorder.Result().Cookies() {
		if c.Name == cookieName {
			assert.Equal(t, -1, c.MaxAge, "cookie %q should have MaxAge=-1 (cleared)", cookieName)
			return
		}
	}
	assert.Fail(t, "cookie %q not found in response", cookieName)
}

// LoginAndGetCookie performs POST /api/login and returns the access_token cookie.
func LoginAndGetCookie(t *testing.T, router *gin.Engine, username, password string) *http.Cookie {
	t.Helper()
	accessCookie, _ := LoginAndGetCookies(t, router, username, password)
	return accessCookie
}

// LoginAndGetCookies performs POST /api/login, returns access_token and refresh_token cookies.
func LoginAndGetCookies(t *testing.T, router *gin.Engine, username, password string) (*http.Cookie, *http.Cookie) {
	t.Helper()

	body := map[string]string{
		"username": username,
		"password": password,
	}
	jsonBody, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var accessCookie, refreshCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "access_token" {
			accessCookie = c
		} else if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}

	require.NotNil(t, accessCookie, "access_token cookie not found in login response")
	return accessCookie, refreshCookie
}

// MakeAuthRequest builds an HTTP request with the access cookie attached.
func MakeAuthRequest(t *testing.T, router *gin.Engine, method, path string,
	body io.Reader, accessCookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessCookie != nil {
		req.AddCookie(accessCookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// MakeAuthRequestJSON is a convenience variant that marshals a JSON body.
func MakeAuthRequestJSON(t *testing.T, router *gin.Engine, method, path string,
	bodyJSON interface{}, accessCookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if bodyJSON != nil {
		jsonBytes, err := json.Marshal(bodyJSON)
		require.NoError(t, err)
		body = bytes.NewReader(jsonBytes)
	}
	return MakeAuthRequest(t, router, method, path, body, accessCookie)
}
