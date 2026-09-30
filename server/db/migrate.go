// Package db embeds the SQL migrations and applies them once each, in order (taken from Devops-pilot).
package db

import (
	"context"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies each additive migration once, under a PostgreSQL transaction
// advisory lock. Existing migration contents are immutable after application.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	names, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(728134098)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(name text PRIMARY KEY, digest bytea NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	for _, name := range names {
		body, err := migrations.ReadFile(name)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(body)
		var old []byte
		err = tx.QueryRow(ctx, `SELECT digest FROM schema_migrations WHERE name=$1`, name).Scan(&old)
		if err == nil {
			if string(old) != string(digest[:]) {
				return fmt.Errorf("migration changed after application: %s", name)
			}
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err = tx.Exec(ctx, string(body), pgx.QueryExecModeSimpleProtocol); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(name,digest) VALUES($1,$2)`, name, digest[:]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
