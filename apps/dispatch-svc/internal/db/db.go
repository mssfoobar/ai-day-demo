// Package db opens the PostgreSQL connection and applies the embedded migrations.
package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/migrations"

	// pgx's database/sql driver, registered as "pgx".
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open connects to PostgreSQL and verifies the connection is usable.
func Open(ctx context.Context, dsn string) (*sqlx.DB, error) {
	pool, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// Migrate applies every embedded migration that has not run yet, in filename order,
// recording each in <schema>.schema_migration.
//
// Embedded and applied on start so no external migration tool is a prerequisite for the
// workshop. Each migration runs in its own transaction, so a failure leaves the previous
// ones applied and the failing one rolled back.
func Migrate(ctx context.Context, pool *sqlx.DB, schema string) error {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	// The ledger lives in the service's own schema, so a shared database keeps one
	// ledger per service rather than a single contended table.
	if _, err := pool.ExecContext(ctx, fmt.Sprintf(
		`CREATE SCHEMA IF NOT EXISTS %s;
		 CREATE TABLE IF NOT EXISTS %s.schema_migration (
		     name       text        NOT NULL PRIMARY KEY,
		     applied_at timestamptz NOT NULL DEFAULT now()
		 );`, schema, schema)); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	for _, name := range names {
		var applied bool
		if err := pool.GetContext(ctx, &applied, fmt.Sprintf(
			`SELECT EXISTS (SELECT 1 FROM %s.schema_migration WHERE name = $1)`, schema),
			name); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if applied {
			continue
		}

		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		stmt := strings.ReplaceAll(string(body), "{{SCHEMA}}", schema)

		tx, err := pool.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			`INSERT INTO %s.schema_migration (name) VALUES ($1)`, schema), name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
