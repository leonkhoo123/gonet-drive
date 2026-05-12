package middleware

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"go-file-server/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jwtTestConfig() *config.CloudConfig {
	return &config.CloudConfig{
		Server: config.ServerConfig{
			AppEnv:         "local",
			FileRoot:       "/tmp/test",
			ListenAddr:     ":0",
			AllowedOrigins: []string{"*"},
		},
		Auth: config.AuthConfig{
			AppJwt:             "ON",
			JwtSecret:          "test-secret-key-for-testing-only",
			AdminUser:          "admin",
			AdminPass:          "admin123",
			CookieAccessToken:  "access_token",
			CookieRefreshToken: "refresh_token",
			CookieMfaPending:   "mfa_pending",
			CookieShareJwt:     "shareJwt",
			AccessTokenMaxAge:  15 * time.Minute,
			RefreshTokenMaxAge: 7 * 24 * time.Hour,
			MfaPendingMaxAge:   5 * time.Minute,
			ShareJwtMaxAge:     7 * 24 * time.Hour,
		},
		Defaults: config.AppDefaults{
			ServiceName:     "GoNet Drive Test",
			UploadChunkSize: "5",
			StorageLimit:    "20480",
		},
	}
}

func TestGenerateAccessToken_Success(t *testing.T) {
	cfg := jwtTestConfig()

	token, err := GenerateAccessToken("testuser", 1, cfg, false, "family-001")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsed, err := ValidateTokenString(token, cfg)
	require.NoError(t, err)
	assert.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(*AccessTokenClaims)
	require.True(t, ok)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, 1, claims.TokenVersion)
	assert.False(t, claims.IsPreAuth)
	assert.Equal(t, "family-001", claims.FamilyID)
}

func TestGenerateAccessToken_PreAuth(t *testing.T) {
	cfg := jwtTestConfig()

	token, err := GenerateAccessToken("testuser", 1, cfg, true, "")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsed, err := ValidateTokenString(token, cfg)
	require.NoError(t, err)

	claims, ok := parsed.Claims.(*AccessTokenClaims)
	require.True(t, ok)
	assert.True(t, claims.IsPreAuth)

	expectedExpiry := time.Now().Add(cfg.Auth.MfaPendingMaxAge)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 2*time.Second)
}

func TestValidateTokenString_Valid(t *testing.T) {
	cfg := jwtTestConfig()

	token, err := GenerateAccessToken("testuser", 1, cfg, false, "")
	require.NoError(t, err)

	parsed, err := ValidateTokenString(token, cfg)
	require.NoError(t, err)
	assert.True(t, parsed.Valid)
}

func TestValidateTokenString_Expired(t *testing.T) {
	cfg := jwtTestConfig()
	cfg.Auth.AccessTokenMaxAge = -1 * time.Minute

	token, err := GenerateAccessToken("testuser", 1, cfg, false, "")
	require.NoError(t, err)

	_, err = ValidateTokenString(token, cfg)
	assert.Error(t, err)
}

func TestValidateTokenString_WrongAlgorithm(t *testing.T) {
	claims := AccessTokenClaims{
		Username:     "testuser",
		TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        "test-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secret := []byte("test-secret-key-for-testing-only")
	validToken, _ := token.SignedString(secret)

	parsed, err := jwt.ParseWithClaims(validToken, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return nil, nil
	})
	assert.Error(t, err)
	assert.False(t, parsed.Valid)
}

func TestValidateTokenString_WrongSecret(t *testing.T) {
	cfg := jwtTestConfig()

	badCfg := jwtTestConfig()
	badCfg.Auth.JwtSecret = "a-different-secret-key"

	token, err := GenerateAccessToken("testuser", 1, cfg, false, "")
	require.NoError(t, err)

	_, err = ValidateTokenString(token, badCfg)
	assert.Error(t, err)
}

func TestGenerateRefreshToken_Length(t *testing.T) {
	token := GenerateRefreshToken()
	assert.NotEmpty(t, token)

	decoded, err := base64.URLEncoding.DecodeString(token)
	assert.NoError(t, err)
	assert.Len(t, decoded, 32)
}

func TestHashToken_Deterministic(t *testing.T) {
	input := "my-test-refresh-token"

	hash1 := HashToken(input)
	hash2 := HashToken(input)

	assert.Equal(t, hash1, hash2)
	assert.NotEmpty(t, hash1)
}

func TestHashToken_DifferentInputs(t *testing.T) {
	hash1 := HashToken("token-a")
	hash2 := HashToken("token-b")

	assert.NotEqual(t, hash1, hash2)
}

func TestHashToken_NotEmpty(t *testing.T) {
	hash := HashToken("some-token")
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestHashToken_EmptyInput(t *testing.T) {
	hash := HashToken("")
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

func TestHashToken_LongInput(t *testing.T) {
	longInput := strings.Repeat("x", 10*1024)
	hash := HashToken(longInput)
	assert.Len(t, hash, 64)
}

func TestHashToken_Unicode(t *testing.T) {
	hash1 := HashToken("こんにちは世界")
	hash2 := HashToken("こんにちは世界")

	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64)

	hash3 := HashToken("😀🎉🚀")
	hash4 := HashToken("😀🎉🚀")
	assert.Equal(t, hash3, hash4)
	assert.Len(t, hash3, 64)
}
