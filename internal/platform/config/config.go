package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

type Config struct {
	Env             Environment
	HTTPAddr        string
	AllowedOrigins  []string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	// APIDocsEnabled serves Swagger UI and Scalar. Off by default in production.
	APIDocsEnabled bool
	Database       Database
}

type Database struct {
	// URL holds the password, so it must never be logged.
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Load reads configuration from environment variables and reports every invalid value at once.
func Load() (Config, error) {
	cfg := Config{
		Env:            Environment(get("APP_ENV", string(Development))),
		HTTPAddr:       get("HTTP_ADDR", ":8080"),
		AllowedOrigins: splitList(get("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
	}

	var errs []error

	switch cfg.Env {
	case Development, Staging, Production:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be development, staging, or production, got %q", cfg.Env))
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}

	cfg.ShutdownTimeout = positiveDuration("SHUTDOWN_TIMEOUT", 10*time.Second, &errs)

	cfg.APIDocsEnabled = cfg.Env != Production
	if raw := os.Getenv("API_DOCS_ENABLED"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf("API_DOCS_ENABLED must be true or false, got %q", raw))
		} else {
			cfg.APIDocsEnabled = enabled
		}
	}

	cfg.Database = Database{
		URL:             strings.TrimSpace(os.Getenv("DATABASE_URL")),
		MaxOpenConns:    positiveInt("DB_MAX_OPEN_CONNS", 20, &errs),
		MaxIdleConns:    positiveInt("DB_MAX_IDLE_CONNS", 5, &errs),
		ConnMaxLifetime: positiveDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute, &errs),
		ConnMaxIdleTime: positiveDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute, &errs),
	}
	switch {
	case cfg.Database.URL == "":
		errs = append(errs, errors.New("DATABASE_URL is required, for example postgres://user:pass@localhost:5432/medpath"))
	case !strings.HasPrefix(cfg.Database.URL, "postgres://") && !strings.HasPrefix(cfg.Database.URL, "postgresql://"):
		errs = append(errs, errors.New("DATABASE_URL must start with postgres:// or postgresql://"))
	}
	if cfg.Database.MaxIdleConns > cfg.Database.MaxOpenConns {
		errs = append(errs, errors.New("DB_MAX_IDLE_CONNS cannot be larger than DB_MAX_OPEN_CONNS"))
	}

	if cfg.Env == Production && len(cfg.AllowedOrigins) == 0 {
		errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS is required in production"))
	}

	return cfg, errors.Join(errs...)
}

func (c Config) IsProduction() bool { return c.Env == Production }

func get(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return fallback
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func positiveInt(key string, fallback int, errs *[]error) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a positive whole number, got %q", key, raw))
		return fallback
	}
	return n
}

func positiveDuration(key string, fallback time.Duration, errs *[]error) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a positive duration such as 10s, got %q", key, raw))
		return fallback
	}
	return d
}
