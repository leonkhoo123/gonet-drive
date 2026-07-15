package middleware

import (
	"database/sql"
	"fmt"
	"go-file-server/internal/config"
	"go-file-server/internal/httpx"
	"go-file-server/internal/repository"
	"go-file-server/internal/state"
	"go-file-server/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	gonetjwt "github.com/leonkhoo123/gonet-auth/jwt"
)

type ShareTokenClaims struct {
	ShareID        string
	AuthorizedPath string
	Authority      string
}

// GenerateShareJWT signs a share token with the library-managed JWT secret.
// The token carries the share ID, authorized path and authority as custom claims.
func GenerateShareJWT(jwtSvc *gonetjwt.Service, shareID, authorizedPath, authority string, d time.Duration) (string, error) {
	return jwtSvc.GenerateCustomToken(map[string]any{
		"share_id":        shareID,
		"authorized_path": authorizedPath,
		"authority":       authority,
	}, d)
}

// ValidateShareToken verifies a share token against the library-managed secret
// (with key-rotation fallback) and extracts the share claims.
func ValidateShareToken(jwtSvc *gonetjwt.Service, tokenStr string) (*ShareTokenClaims, error) {
	parsed, err := jwtSvc.ValidateCustomToken(tokenStr)
	if err != nil {
		return nil, err
	}
	mc, ok := parsed.Claims.(*jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}
	shareID, _ := (*mc)["share_id"].(string)
	path, _ := (*mc)["authorized_path"].(string)
	authority, _ := (*mc)["authority"].(string)
	if shareID == "" || path == "" || authority == "" {
		return nil, fmt.Errorf("missing required share claims")
	}
	return &ShareTokenClaims{ShareID: shareID, AuthorizedPath: path, Authority: authority}, nil
}

// ShareAuthMiddleware verifies shareJwt token on share endpoints.
// After JWT validation, checks cache then DB to ensure the share row
// still exists and is not blocked.
func ShareAuthMiddleware(cfg *config.CloudConfig, shareRepo repository.SharingRepository, jwtSvc *gonetjwt.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		shareID := c.GetHeader("X-Share-Id")
		if shareID == "" {
			shareID = c.Query("share_id")
		}
		if shareID == "" {
			httpx.Abort(c, http.StatusBadRequest, "missing share_id in request")
			return
		}

		tokenStr, err := util.GetShareJwt(c, cfg, shareID)
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				httpx.Abort(c, http.StatusUnauthorized, "missing share token")
				return
			}
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		claims, err := ValidateShareToken(jwtSvc, tokenStr)
		if err != nil {
			httpx.Abort(c, http.StatusUnauthorized, "invalid or expired share token")
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
					httpx.Abort(c, http.StatusUnauthorized, "share link no longer valid")
					return
				}
				httpx.Abort(c, http.StatusInternalServerError, "failed to validate share")
				return
			}
			exists = true
			blocked = share.Blocked
			state.SetShareStatus(claims.ShareID, exists, blocked)
		}

		if !exists {
			httpx.Abort(c, http.StatusUnauthorized, "share link no longer valid")
			return
		}
		if blocked {
			httpx.Abort(c, http.StatusUnauthorized, "share link no longer valid")
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
		if !exists {
			httpx.Abort(c, http.StatusForbidden, "share link does not have modify authority")
			return
		}
		authStr, ok := authority.(string)
		if !ok || authStr != "modify" {
			httpx.Abort(c, http.StatusForbidden, "share link does not have modify authority")
			return
		}
		c.Next()
	}
}
