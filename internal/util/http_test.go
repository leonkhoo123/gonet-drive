package util

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBaseURL_BasicHTTP(t *testing.T) {
	req := &http.Request{
		Host:   "localhost",
		Header: http.Header{},
	}
	assert.Equal(t, "http://localhost", GetBaseURL(req))
}

func TestGetBaseURL_TLS(t *testing.T) {
	req := &http.Request{
		Host:   "example.com",
		Header: http.Header{},
		TLS:    &tls.ConnectionState{},
	}
	assert.Equal(t, "https://example.com", GetBaseURL(req))
}

func TestGetBaseURL_XForwardedProto(t *testing.T) {
	req := &http.Request{
		Host: "hostname",
		Header: http.Header{
			"X-Forwarded-Proto": []string{"https"},
		},
	}
	assert.Equal(t, "https://hostname", GetBaseURL(req))
}

func TestGetBaseURL_CloudflareVisitor(t *testing.T) {
	req := &http.Request{
		Host: "hostname",
		Header: http.Header{
			"Cf-Visitor": []string{`{"scheme":"https"}`},
		},
	}
	assert.Equal(t, "https://hostname", GetBaseURL(req))
}

func TestGetBaseURL_XForwardedHost(t *testing.T) {
	req := &http.Request{
		Host: "internal.example.com",
		Header: http.Header{
			"X-Forwarded-Host": []string{"custom.example.com"},
		},
	}
	assert.Equal(t, "http://custom.example.com", GetBaseURL(req))
}

func TestGetBaseURL_XForwardedHostEmpty(t *testing.T) {
	req := &http.Request{
		Host: "fallback.example.com",
		Header: http.Header{
			"X-Forwarded-Host": []string{""},
		},
	}
	assert.Equal(t, "http://fallback.example.com", GetBaseURL(req))
}

func TestGetBaseURL_HTTPOverXForwarded(t *testing.T) {
	req := &http.Request{
		Host: "hostname",
		Header: http.Header{
			"X-Forwarded-Proto": []string{"http"},
		},
	}
	assert.Equal(t, "http://hostname", GetBaseURL(req))
}
