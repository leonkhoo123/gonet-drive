package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
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

type AccessTokenClaims struct {
	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"`
	IsPreAuth    bool   `json:"is_pre_auth,omitempty"`
	FamilyID     string `json:"family_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateAccessToken returns a signed token with preset expiry from util
func GenerateAccessToken(username string, tokenVersion int, cfg *config.CloudConfig, isPreAuth bool, familyID string) (string, error) {
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
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
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

func ValidateTokenString(tokenStr string, cfg *config.CloudConfig) (*jwt.Token, error) {
	var jwtSecret = []byte(cfg.Auth.JwtSecret)
	return jwt.ParseWithClaims(tokenStr, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
}

// JWTAuthMiddleware verifies JWT token on protected routes
func JWTAuthMiddleware(cfg *config.CloudConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := util.GetAccessToken(c, cfg)
		if err != nil {
			// Fallback to legacy token name or auth header
			tokenStr, err = util.GetLegacyToken(c, cfg.Auth.TokenName)
			if err != nil {
				authHeader := c.GetHeader("Authorization")
				if authHeader == "" {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
					return
				}
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		token, err := ValidateTokenString(tokenStr, cfg)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(*AccessTokenClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		if claims.IsPreAuth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "pre-auth token not allowed"})
			return
		}

		if claims.FamilyID != "" {
			if _, revoked := RevokedSessionsCache.Get(claims.FamilyID); revoked {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session revoked"})
				return
			}
		}

		// Check token_version against DB
		var currentVersion int
		var mfaEnabled bool
		var mfaMandatory bool
		err = config.DB.QueryRow("SELECT token_version, mfa_enabled, mfa_mandatory FROM users WHERE username = ?", claims.Username).Scan(&currentVersion, &mfaEnabled, &mfaMandatory)
		if err != nil {
			if err == sql.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		if claims.TokenVersion != currentVersion {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
			return
		}

		if !mfaEnabled && mfaMandatory {
			path := c.Request.URL.Path
			if path != "/api/user/me" && path != "/api/user/mfa/setup" && path != "/api/user/mfa/enable" && path != "/api/logout" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "mfa_setup_required"})
				return
			}
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}
