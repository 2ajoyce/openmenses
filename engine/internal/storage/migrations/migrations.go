// Package migrations provides the schema migration runner for the SQLite backend.
// Migrations run inside a transaction and are tracked in the schema_migrations table.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

// Migration is a single versioned DDL change.
type Migration struct {
	Version int
	SQL     string
}

// All returns the ordered list of all migrations.
func All() []Migration {
	return []Migration{
		{Version: 1, SQL: v1SQL},
	}
}

// Run applies all pending migrations in order. It is safe to call multiple times;
// already-applied migrations are skipped (idempotent).
func Run(ctx context.Context, db *sql.DB) error {
	if err := ensureTable(ctx, db); err != nil {
		return fmt.Errorf("migrations: ensure table: %w", err)
	}
	for _, m := range All() {
		if err := apply(ctx, db, m); err != nil {
			return fmt.Errorf("migrations: apply v%d: %w", m.Version, err)
		}
	}
	return nil
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		)`)
	return err
}

func apply(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, m.Version).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return tx.Commit()
	}

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("execute DDL: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, m.Version); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return tx.Commit()
}

// v1SQL creates all tables for the initial schema.
const v1SQL = `
CREATE TABLE IF NOT EXISTS user_profiles (
	id            TEXT PRIMARY KEY,
	data          BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS bleeding_observations (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	timestamp  TEXT NOT NULL,
	data       BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_bleeding_user_ts ON bleeding_observations (user_id, timestamp);

CREATE TABLE IF NOT EXISTS symptom_observations (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	timestamp  TEXT NOT NULL,
	data       BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_symptom_user_ts ON symptom_observations (user_id, timestamp);

CREATE TABLE IF NOT EXISTS mood_observations (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	timestamp  TEXT NOT NULL,
	data       BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_mood_user_ts ON mood_observations (user_id, timestamp);

CREATE TABLE IF NOT EXISTS medications (
	id      TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	data    BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_medications_user ON medications (user_id);

CREATE TABLE IF NOT EXISTS medication_events (
	id            TEXT PRIMARY KEY,
	user_id       TEXT NOT NULL,
	medication_id TEXT NOT NULL,
	timestamp     TEXT NOT NULL,
	data          BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_med_events_user_ts    ON medication_events (user_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_med_events_medication ON medication_events (medication_id);

CREATE TABLE IF NOT EXISTS cycles (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL,
	start_date TEXT NOT NULL,
	end_date   TEXT,
	data       BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cycles_user ON cycles (user_id);

CREATE TABLE IF NOT EXISTS phase_estimates (
	id      TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	date    TEXT NOT NULL,
	data    BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_phase_user_date ON phase_estimates (user_id, date);

CREATE TABLE IF NOT EXISTS predictions (
	id      TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	data    BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_predictions_user ON predictions (user_id);

CREATE TABLE IF NOT EXISTS insights (
	id      TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	data    BLOB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_insights_user ON insights (user_id);
`
