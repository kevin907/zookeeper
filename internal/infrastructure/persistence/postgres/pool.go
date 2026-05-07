// Package postgres is the Postgres + JSONB Repository adapter backed by pgxpool.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevin907/zookeeper/internal/platform/config"
)

// ErrMissingURL is returned when the config does not carry a POSTGRES_URL.
var ErrMissingURL = errors.New("POSTGRES_URL is required")

// NewPool constructs a pgxpool.Pool from config. Credentials in POSTGRES_URL
// are never echoed in error messages; the caller receives a scrubbed error.
func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	if cfg.PostgresURL == "" {
		return nil, ErrMissingURL
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", scrubErr(err))
	}
	poolCfg.MaxConns = cfg.PostgresMaxConns
	poolCfg.MinConns = cfg.PostgresMinConns
	poolCfg.MaxConnLifetime = cfg.PostgresMaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.PostgresMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open pgxpool: %w", scrubErr(err))
	}
	return pool, nil
}

// scrubErr returns a generic error rather than letting driver-level messages
// potentially leak connection-string fragments.
func scrubErr(_ error) error {
	return errors.New("postgres driver error (scrubbed)")
}
