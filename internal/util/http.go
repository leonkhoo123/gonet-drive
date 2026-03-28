package util

import (
	"fmt"
	"net/http"
)

// GetBaseURL constructs the absolute base URL (e.g., https://drive.leonkhoo.com)
// by inspecting the incoming HTTP request headers, specifically looking for
// proxies like Nginx or Cloudflare Tunnels.
func GetBaseURL(r *http.Request) string {
	scheme := "http"

	// Check if forwarded by proxy via HTTPS
	if r.Header.Get("X-Forwarded-Proto") == "https" || r.TLS != nil {
		scheme = "https"
	} else if r.Header.Get("Cf-Visitor") != "" {
		// Cloudflare specific check (e.g., {"scheme":"https"})
		// Simplified check since Cloudflare tunnels typically terminate HTTPS
		scheme = "https"
	}

	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}
