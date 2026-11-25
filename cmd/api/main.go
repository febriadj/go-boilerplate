package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"medpath-server/internal/feature/health"
	"medpath-server/internal/platform/apidocs"
	"medpath-server/internal/platform/config"
	"medpath-server/internal/platform/database"
	"medpath-server/internal/platform/httpserver"
)

// Set at build time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

// @title			Medpath API
// @version		0.1.0
// @description	Clinical pathway service that runs alongside SIMRS.
// @BasePath		/
// @schemes		http https
func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		// Config may have failed before the app logger exists, so log to stderr directly.
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.Database, logger, cfg.LogLevel <= slog.LevelDebug)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(db); err != nil {
			logger.Error("close database", slog.Any("error", err))
		}
	}()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	router := httpserver.NewRouter(cfg, logger)
	health.Register(router, version, sqlDB, logger)
	if cfg.APIDocsEnabled {
		apidocs.Register(router, version)
	}

	srv := httpserver.NewServer(cfg.HTTPAddr, router)

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", slog.String("addr", cfg.HTTPAddr), slog.String("env", string(cfg.Env)), slog.String("version", version))
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	logger.Info("shutting down", slog.Duration("timeout", cfg.ShutdownTimeout))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.Env == config.Development {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// healthcheck asks the running server for /readyz. The production image has no shell or curl,
// so the container healthcheck runs the binary itself: medpath-server healthcheck.
// Only the port comes from HTTP_ADDR; the host is always loopback, so it cannot be pointed elsewhere.
func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	_, rawPort, err := net.SplitHostPort(addr)
	if err != nil {
		return 1
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 1 || port > 65535 {
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	url := fmt.Sprintf("http://127.0.0.1:%d/readyz", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) //nolint:gosec // G704: host is fixed to 127.0.0.1, only a validated port comes from env
	if err != nil {
		return 1
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: same loopback-only URL as above
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
