package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"APP_ENV", "HTTP_ADDR", "CORS_ALLOWED_ORIGINS", "LOG_LEVEL", "SHUTDOWN_TIMEOUT"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != Development || cfg.HTTPAddr != ":8080" || cfg.LogLevel != slog.LevelInfo || cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "http://localhost:3000" {
		t.Fatalf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
}

func TestLoadParsesValues(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example, https://b.example ,")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.IsProduction() || cfg.HTTPAddr != ":9000" || cfg.LogLevel != slog.LevelDebug || cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if strings.Join(cfg.AllowedOrigins, "|") != "https://a.example|https://b.example" {
		t.Fatalf("AllowedOrigins = %v", cfg.AllowedOrigins)
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("SHUTDOWN_TIMEOUT", "-1s")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want errors")
	}
	for _, want := range []string{"APP_ENV", "LOG_LEVEL", "SHUTDOWN_TIMEOUT"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}
