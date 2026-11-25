package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"medpath-server/internal/platform/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	connectTimeout = 5 * time.Second
	slowQuery      = 200 * time.Millisecond
)

// Open connects to PostgreSQL and fails fast if the database cannot be reached.
// With logSQL, every query is logged; otherwise only errors and slow queries are.
func Open(ctx context.Context, cfg config.Database, logger *slog.Logger, logSQL bool) (*gorm.DB, error) {
	level := gormlogger.Warn
	if logSQL {
		level = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		Logger: gormlogger.NewSlogLogger(logger.With(slog.String("component", "gorm")), gormlogger.Config{
			SlowThreshold: slowQuery,
			// Log SQL with $1 placeholders only: bound values can be patient data.
			ParameterizedQueries:      true,
			IgnoreRecordNotFoundError: true,
			LogLevel:                  level,
		}),
		// Turns driver errors such as unique violations into gorm.ErrDuplicatedKey.
		TranslateError: true,
		NowFunc:        func() time.Time { return time.Now().UTC() },
		// The ping below has a timeout and a single clear error; GORM's own ping has neither.
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database handle: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("reach database: %w", err)
	}
	return db, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
