package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

const testDatabaseURL = "postgres://medpath:secret@localhost:5432/medpath"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"APP_ENV", "HTTP_ADDR", "CORS_ALLOWED_ORIGINS", "LOG_LEVEL", "SHUTDOWN_TIMEOUT", "API_DOCS_ENABLED",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("DATABASE_URL", testDatabaseURL)

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
	db := cfg.Database
	if db.URL != testDatabaseURL || db.MaxOpenConns != 20 || db.MaxIdleConns != 5 ||
		db.ConnMaxLifetime != 30*time.Minute || db.ConnMaxIdleTime != 5*time.Minute {
		t.Fatalf("unexpected database defaults: %+v", db)
	}
}

func TestLoadParsesValues(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example, https://b.example ,")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")
	t.Setenv("DATABASE_URL", testDatabaseURL)
	t.Setenv("DB_MAX_OPEN_CONNS", "50")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")

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
	if cfg.Database.MaxOpenConns != 50 || cfg.Database.MaxIdleConns != 10 {
		t.Fatalf("unexpected pool size: %+v", cfg.Database)
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("LOG_LEVEL", "loud")
	t.Setenv("SHUTDOWN_TIMEOUT", "-1s")
	t.Setenv("DATABASE_URL", "mysql://root@localhost/medpath")
	t.Setenv("DB_MAX_OPEN_CONNS", "zero")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want errors")
	}
	for _, want := range []string{"APP_ENV", "LOG_LEVEL", "SHUTDOWN_TIMEOUT", "DATABASE_URL", "DB_MAX_OPEN_CONNS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}

func TestAPIDocsDefaultsByEnvironment(t *testing.T) {
	cases := []struct {
		env, flag string
		want      bool
	}{
		{"development", "", true},
		{"staging", "", true},
		{"production", "", false},
		{"production", "true", true},
		{"development", "false", false},
	}
	for _, tc := range cases {
		t.Run(tc.env+"/"+tc.flag, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("API_DOCS_ENABLED", tc.flag)
			t.Setenv("DATABASE_URL", testDatabaseURL)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.APIDocsEnabled != tc.want {
				t.Fatalf("APIDocsEnabled = %v, want %v", cfg.APIDocsEnabled, tc.want)
			}
		})
	}
}

func TestDatabaseURLIsRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("Load() error = %v, want DATABASE_URL is required", err)
	}
}
