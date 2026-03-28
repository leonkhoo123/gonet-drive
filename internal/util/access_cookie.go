package util

import (
	"net/http"

	"go-file-server/internal/config"

	"github.com/gin-gonic/gin"
)

// getSecureMode checks if we're running locally.
func getSecureMode(cfg *config.CloudConfig) bool {
	if cfg.Server.AppEnv == "local" || cfg.Server.AppEnv == "dev" {
		return false
	} else {
		return true
	}
}

// SetAccessToken sets the access token cookie with preset expiration
func SetAccessToken(c *gin.Context, cfg *config.CloudConfig, token string) {
	setCookie(c, cfg.Auth.CookieAccessToken, token, int(cfg.Auth.AccessTokenMaxAge.Seconds()), http.SameSiteStrictMode, getSecureMode(cfg))
}

func ClearAccessToken(c *gin.Context, cfg *config.CloudConfig) {
	clearCookie(c, cfg.Auth.CookieAccessToken, http.SameSiteStrictMode, getSecureMode(cfg))
}

func GetAccessToken(c *gin.Context, cfg *config.CloudConfig) (string, error) {
	return c.Cookie(cfg.Auth.CookieAccessToken)
}

// SetRefreshToken sets the refresh token cookie with preset expiration
func SetRefreshToken(c *gin.Context, cfg *config.CloudConfig, token string) {
	setCookie(c, cfg.Auth.CookieRefreshToken, token, int(cfg.Auth.RefreshTokenMaxAge.Seconds()), http.SameSiteStrictMode, getSecureMode(cfg))
}

func ClearRefreshToken(c *gin.Context, cfg *config.CloudConfig) {
	clearCookie(c, cfg.Auth.CookieRefreshToken, http.SameSiteStrictMode, getSecureMode(cfg))
}

func GetRefreshToken(c *gin.Context, cfg *config.CloudConfig) (string, error) {
	return c.Cookie(cfg.Auth.CookieRefreshToken)
}

// SetMfaPendingToken sets the MFA pending cookie with preset expiration
func SetMfaPendingToken(c *gin.Context, cfg *config.CloudConfig, token string) {
	setCookie(c, cfg.Auth.CookieMfaPending, token, int(cfg.Auth.MfaPendingMaxAge.Seconds()), http.SameSiteStrictMode, getSecureMode(cfg))
}

func ClearMfaPendingToken(c *gin.Context, cfg *config.CloudConfig) {
	clearCookie(c, cfg.Auth.CookieMfaPending, http.SameSiteStrictMode, getSecureMode(cfg))
}

func GetMfaPendingToken(c *gin.Context, cfg *config.CloudConfig) (string, error) {
	return c.Cookie(cfg.Auth.CookieMfaPending)
}

// SetShareJwt sets the share JWT cookie.
// It takes maxAge as an argument because share links have dynamic expirations,
// but usually it does not exceed ShareJwtMaxAge.
func SetShareJwt(c *gin.Context, cfg *config.CloudConfig, token string, maxAge int, shareID string) {
	// Use Lax mode for sharing so it can be sent on initial top-level navigation (like opening a share link)
	setCookie(c, cfg.Auth.CookieShareJwt+"_"+shareID, token, maxAge, http.SameSiteLaxMode, getSecureMode(cfg))
}

func ClearShareJwt(c *gin.Context, cfg *config.CloudConfig, shareID string) {
	clearCookie(c, cfg.Auth.CookieShareJwt+"_"+shareID, http.SameSiteLaxMode, getSecureMode(cfg))
}

func GetShareJwt(c *gin.Context, cfg *config.CloudConfig, shareID string) (string, error) {
	return c.Cookie(cfg.Auth.CookieShareJwt + "_" + shareID)
}

// ClearLegacyToken clears the legacy token
func ClearLegacyToken(c *gin.Context, tokenName string) {
	// For legacy token we default to true, or we could pass config.AppConfig.Server.AppEnv != "local"
	clearCookie(c, tokenName, http.SameSiteStrictMode, config.AppConfig.Server.AppEnv != "local")
}

// GetLegacyToken gets the legacy token
func GetLegacyToken(c *gin.Context, tokenName string) (string, error) {
	return c.Cookie(tokenName)
}

// setCookie is a private helper
func setCookie(c *gin.Context, name, value string, maxAge int, sameSite http.SameSite, secure bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   maxAge,
		Path:     "/",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	}
	http.SetCookie(c.Writer, cookie)
}

// clearCookie is a private helper
func clearCookie(c *gin.Context, name string, sameSite http.SameSite, secure bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Domain:   "",
		Secure:   secure,
		HttpOnly: true,
		SameSite: sameSite,
	}
	http.SetCookie(c.Writer, cookie)
}
