// Package migrate wraps goose to run embedded SQL migrations against a pgxpool.
package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/kevin907/zookeeper/migrations"
)

const dialect = "postgres"

// Up applies every pending migration embedded in the binary.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	return run(ctx, pool, func(db *sql.DB) error {
		return goose.UpContext(ctx, db, ".")
	})
}

// Down rolls back a single migration step; used by the integration test.
func Down(ctx context.Context, pool *pgxpool.Pool) error {
	return run(ctx, pool, func(db *sql.DB) error {
		return goose.DownContext(ctx, db, ".")
	})
}

func run(ctx context.Context, pool *pgxpool.Pool, op func(*sql.DB) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := op(db); err != nil {
		return fmt.Errorf("goose migration: %w", err)
	}
	return nil
}
