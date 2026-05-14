package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"go-file-server/internal/config"
	"go-file-server/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
)

var RevokedSessionsCache = cache.New(20*time.Minute, 30*time.Minute)
var UserRoleCache = cache.New(5*time.Minute, 10*time.Minute)

type AccessTokenClaims struct {
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	IsPreAuth    bool   `json:"is_pre_auth,omitempty"`
	FamilyID     string `json:"family_id,omitempty"`
	Role         string `json:"role"`
	IsSuperAdmin bool   `json:"is_super_admin"`
	Type         string `json:"type"`
	jwt.RegisteredClaims
}

type UserRoleInfo struct {
	Role         string
	IsSuperAdmin bool
	TokenVersion int
	MfaEnabled   bool
	MfaMandatory bool
}

// GetUserRoleInfo fetches user role info from cache or DB. The caller should handle errors.
func GetUserRoleInfo(username string, cfg *config.CloudConfig) (*UserRoleInfo, error) {
	if cached, found := UserRoleCache.Get(username); found {
		return cached.(*UserRoleInfo), nil
	}

	var info UserRoleInfo
	err := config.DB.QueryRow(
		"SELECT role, token_version, mfa_enabled, mfa_mandatory FROM users WHERE username = ?",
		username,
	).Scan(&info.Role, &info.TokenVersion, &info.MfaEnabled, &info.MfaMandatory)
	if err != nil {
		return nil, err
	}

	info.IsSuperAdmin = username == cfg.Auth.AdminUser

	UserRoleCache.Set(username, &info, cache.DefaultExpiration)
	return &info, nil
}

// ClearUserRoleCache removes a user's entry from the role cache.
// Call this whenever a user's role, token_version, or MFA settings change.
func ClearUserRoleCache(username string) {
	UserRoleCache.Delete(username)
}

// GenerateAccessToken returns a signed token with preset expiry from util
func GenerateAccessToken(username string, tokenVersion int, role string, isSuperAdmin bool, cfg *config.CloudConfig, isPreAuth bool, familyID string) (string, error) {
	var jwtSecret = []byte(cfg.Auth.JwtSecret)

	duration := cfg.Auth.AccessTokenMaxAge
	if isPreAuth {
		duration = cfg.Auth.MfaPendingMaxAge
	}

	claims := AccessTokenClaims{
		Username:     username,
		TokenVersion: tokenVersion,
		IsPreAuth:    isPreAuth,
		FamilyID:     familyID,
		Role:         role,
		IsSuperAdmin: isSuperAdmin,
		Type:         "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func GenerateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateMobileRefreshToken(username, familyID, deviceID string, duration time.Duration, cfg *config.CloudConfig) (string, error) {
	var jwtSecret = []byte(cfg.Auth.JwtSecret)

	claims := AccessTokenClaims{
		Username: username,
		FamilyID: familyID,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ExtractAndValidateBearerAccess(c *gin.Context, cfg *config.CloudConfig) (*AccessTokenClaims, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		return nil, fmt.Errorf("invalid Authorization format")
	}

	token, err := ValidateTokenString(tokenStr, cfg)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid or expired token")
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.IsPreAuth {
		return nil, fmt.Errorf("pre-auth token not allowed")
	}

	return claims, nil
}

func ValidateTokenString(tokenStr string, cfg *config.CloudConfig) (*jwt.Token, error) {
	var jwtSecret = []byte(cfg.Auth.JwtSecret)
	return jwt.ParseWithClaims(tokenStr, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
}

// JWTAuthMiddleware verifies JWT token on protected routes.
// Accepts authentication via either Authorization: Bearer <token> header (mobile) or cookie (web).
// Bearer header takes precedence when present.
// All paths are validated: token expiry, pre-auth rejection, session revocation,
// token_version match, role match (cached from DB), and MFA mandatory enforcement.
func JWTAuthMiddleware(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var claims *AccessTokenClaims

		// 1. Check Bearer header first (mobile app) — uses shared extraction helper
		if c.GetHeader("Authorization") != "" {
			var err error
			claims, err = ExtractAndValidateBearerAccess(c, cfg)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
		} else {
			// 2. Fall back to access_token cookie
			tokenStr, _ := util.GetAccessToken(c, cfg)

			// 3. Fall back to legacy token cookie
			if tokenStr == "" {
				tokenStr, _ = util.GetLegacyToken(c, cfg.Auth.TokenName)
			}

			if tokenStr == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
				return
			}

			token, err := ValidateTokenString(tokenStr, cfg)
			if err != nil || !token.Valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
				return
			}

			var ok bool
			claims, ok = token.Claims.(*AccessTokenClaims)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
				return
			}

			if claims.IsPreAuth {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "pre-auth token not allowed"})
				return
			}
		}

		// Check revoked sessions (fast cache check, no DB)
		if claims.FamilyID != "" {
			if _, revoked := RevokedSessionsCache.Get(claims.FamilyID); revoked {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session revoked"})
				return
			}
		}

		// Get current user info from DB (cached)
		roleInfo, err := GetUserRoleInfo(claims.Username, cfg)
		if err != nil {
			if err == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		// Validate token_version matches current DB value
		if claims.TokenVersion != roleInfo.TokenVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			return
		}

		// Validate role matches current DB value (protects against stale role claims after admin role change)
		if claims.Role != roleInfo.Role || claims.IsSuperAdmin != roleInfo.IsSuperAdmin {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalidated due to role change"})
			return
		}

		// MFA mandatory enforcement
		if !roleInfo.MfaEnabled && roleInfo.MfaMandatory {
			path := c.Request.URL.Path
			if path != "/api/user/me" && path != "/api/user/mfa/setup" && path != "/api/user/mfa/enable" && path != "/api/logout" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "mfa_setup_required"})
				return
			}
		}

		deviceID := c.GetHeader("X-Device-Id")

		c.Set("username", claims.Username)
		c.Set("role", roleInfo.Role)
		c.Set("is_super_admin", roleInfo.IsSuperAdmin)
		c.Set("family_id", claims.FamilyID)
		c.Set("device_id", deviceID)
		c.Set("mfa_enabled", roleInfo.MfaEnabled)
		c.Set("mfa_mandatory", roleInfo.MfaMandatory)
		c.Next()
	}
}
