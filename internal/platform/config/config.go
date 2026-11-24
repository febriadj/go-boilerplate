package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
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
}

// Load reads configuration from environment variables and reports every invalid value at once.
func Load() (Config, error) {
	cfg := Config{
		Env:             Environment(get("APP_ENV", string(Development))),
		HTTPAddr:        get("HTTP_ADDR", ":8080"),
		AllowedOrigins:  splitList(get("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		ShutdownTimeout: 10 * time.Second,
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

	if raw := os.Getenv("SHUTDOWN_TIMEOUT"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d <= 0 {
			errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration such as 10s, got %q", raw))
		} else {
			cfg.ShutdownTimeout = d
		}
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
