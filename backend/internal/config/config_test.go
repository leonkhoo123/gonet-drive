package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------- Finding 3.1: getSecureMode ----------

func TestGetSecureMode_DefaultProduction(t *testing.T) {
	os.Unsetenv("SECURE_MODE")
	defer os.Unsetenv("SECURE_MODE")

	result := getSecureMode("production")
	assert.True(t, result, "should default to true when APP_ENV=production")
}

func TestGetSecureMode_DefaultLocal(t *testing.T) {
	os.Unsetenv("SECURE_MODE")
	defer os.Unsetenv("SECURE_MODE")

	result := getSecureMode("local")
	assert.False(t, result, "should default to false when APP_ENV=local")
}

func TestGetSecureMode_ExplicitTrue(t *testing.T) {
	os.Setenv("SECURE_MODE", "true")
	defer os.Unsetenv("SECURE_MODE")

	assert.True(t, getSecureMode("local"))
}

func TestGetSecureMode_ExplicitOne(t *testing.T) {
	os.Setenv("SECURE_MODE", "1")
	defer os.Unsetenv("SECURE_MODE")

	assert.True(t, getSecureMode("local"))
}

func TestGetSecureMode_ExplicitFalse(t *testing.T) {
	os.Setenv("SECURE_MODE", "false")
	defer os.Unsetenv("SECURE_MODE")

	assert.False(t, getSecureMode("production"))
}

func TestGetSecureMode_ExplicitZero(t *testing.T) {
	os.Setenv("SECURE_MODE", "0")
	defer os.Unsetenv("SECURE_MODE")

	assert.False(t, getSecureMode("production"))
}
