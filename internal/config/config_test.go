package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv_Fallback(t *testing.T) {
	t.Setenv("TEST_KEY", "")
	assert.Equal(t, "default", getEnv("TEST_KEY", "default"))
}

func TestGetEnv_Value(t *testing.T) {
	t.Setenv("TEST_KEY", "value")
	assert.Equal(t, "value", getEnv("TEST_KEY", "default"))
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("APP_PORT", "")
	t.Setenv("DB_HOST", "")
	cfg := LoadConfig()
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
}

func TestLoadConfig_Custom(t *testing.T) {
	t.Setenv("APP_PORT", "9000")
	t.Setenv("SEARCH_ENGINE_ENDPOINT", "http://search.test")
	cfg := LoadConfig()
	assert.Equal(t, "9000", cfg.AppPort)
	assert.Equal(t, "http://search.test", cfg.SearchEngineEndpoint)
}
