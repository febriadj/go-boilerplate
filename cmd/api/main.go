package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"medpath-server/internal/feature/health"
	"medpath-server/internal/platform/apidocs"
	"medpath-server/internal/platform/config"
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
	if err := run(); err != nil {
		slog.Error("server stopped", slog.Any("error", err))
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

	router := httpserver.NewRouter(cfg, logger)
	health.Register(router, version)
	if cfg.APIDocsEnabled {
		apidocs.Register(router, version)
	}

	srv := httpserver.NewServer(cfg.HTTPAddr, router)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
