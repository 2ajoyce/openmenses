package migrations_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/2ajoyce/openmenses/engine/internal/storage/migrations"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRun_CreatesAllTables(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	if err := migrations.Run(ctx, db); err != nil {
		t.Fatal(err)
	}

	tables := []string{
		"schema_migrations",
		"user_profiles",
		"bleeding_observations",
		"symptom_observations",
		"mood_observations",
		"medications",
		"medication_events",
		"cycles",
		"phase_estimates",
		"predictions",
		"insights",
	}
	for _, table := range tables {
		var name string
		err := db.QueryRowContext(ctx,
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestRun_Idempotent(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	if err := migrations.Run(ctx, db); err != nil {
		t.Fatal(err)
	}
	// Running again should not error.
	if err := migrations.Run(ctx, db); err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	var count int
	db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count) //nolint:errcheck
	if count != len(migrations.All()) {
		t.Fatalf("want %d migration rows, got %d", len(migrations.All()), count)
	}
}
