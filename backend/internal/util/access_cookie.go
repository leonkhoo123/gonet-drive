package util

import (
	"net/http"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
)

// getSecureMode checks if we're running locally.
func getSecureMode(cfg *config.CloudConfig) bool {
	return cfg.Server.AppEnv != "local" && cfg.Server.AppEnv != "dev"
}

// SetShareJwt sets the share JWT cookie.
func SetShareJwt(c *gin.Context, cfg *config.CloudConfig, token string, maxAge int, shareID string) {
	cookie := &http.Cookie{
		Name:     cfg.Auth.CookieShareJwt + "_" + shareID,
		Value:    token,
		MaxAge:   maxAge,
		Path:     "/",
		Secure:   getSecureMode(cfg),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

func ClearShareJwt(c *gin.Context, cfg *config.CloudConfig, shareID string) {
	cookie := &http.Cookie{
		Name:     cfg.Auth.CookieShareJwt + "_" + shareID,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   getSecureMode(cfg),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, cookie)
}

func GetShareJwt(c *gin.Context, cfg *config.CloudConfig, shareID string) (string, error) {
	return c.Cookie(cfg.Auth.CookieShareJwt + "_" + shareID)
}
