package middleware

import (
	"database/sql"
	"go-file-server/internal/config"
	"go-file-server/internal/repository"
	"go-file-server/internal/state"
	"go-file-server/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ShareTokenClaims struct {
	ShareID        string `json:"share_id"`
	AuthorizedPath string `json:"authorized_path"`
	Authority      string `json:"authority"`
	jwt.RegisteredClaims
}

// GenerateShareJWT returns a signed token with variable expiry based on link expiry
func GenerateShareJWT(shareID string, authorizedPath string, authority string, duration time.Duration, secret string) (string, error) {
	var jwtSecret = []byte(secret)
	claims := ShareTokenClaims{
		ShareID:        shareID,
		AuthorizedPath: authorizedPath,
		Authority:      authority,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateShareTokenString(tokenStr string, secret string) (*jwt.Token, error) {
	var jwtSecret = []byte(secret)
	return jwt.ParseWithClaims(tokenStr, &ShareTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})
}

// ShareAuthMiddleware verifies shareJwt token on share endpoints.
// After JWT validation, checks cache then DB to ensure the share row
// still exists and is not blocked.
func ShareAuthMiddleware(cfg *config.CloudConfig, shareRepo repository.SharingRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		shareID := c.GetHeader("X-Share-Id")
		if shareID == "" {
			shareID = c.Query("share_id")
		}
		if shareID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing share_id in request"})
			return
		}

		secret := cfg.Auth.JwtSecret
		tokenStr, err := util.GetShareJwt(c, cfg, shareID)
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing share token"})
				return
			}
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		token, err := ValidateShareTokenString(tokenStr, secret)

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired share token"})
			return
		}

		claims, ok := token.Claims.(*ShareTokenClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid share token claims"})
			return
		}

		// Check cache-first, fall back to DB, to block deleted/blocked shares
		// even when the JWT is still valid.
		exists, blocked, found := state.GetShareStatus(claims.ShareID)
		if !found {
			share, err := shareRepo.GetByID(claims.ShareID)
			if err != nil {
				// ErrNoRows means share was deleted
				if err == sql.ErrNoRows {
					state.SetShareStatus(claims.ShareID, false, false)
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "share link no longer valid"})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to validate share"})
				return
			}
			exists = true
			blocked = share.Blocked
			state.SetShareStatus(claims.ShareID, exists, blocked)
		}

		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "share link no longer valid"})
			return
		}
		if blocked {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "share link no longer valid"})
			return
		}

		// Set context variables for the share_file_controller to use
		c.Set("share_id", claims.ShareID)
		c.Set("authorized_path", claims.AuthorizedPath)
		c.Set("authority", claims.Authority)
		c.Next()
	}
}

// ShareModifyAuthorityMiddleware ensures the shareJwt has 'modify' authority
func ShareModifyAuthorityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authority, exists := c.Get("authority")
		if !exists || authority.(string) != "modify" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "share link does not have modify authority"})
			return
		}
		c.Next()
	}
}
