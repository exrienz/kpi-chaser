package storage

import (
	"path/filepath"
	"testing"
)

func TestMigrateAddsKPIHierarchyColumnsToExistingTable(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "migrate-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE kpis (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			quarter TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			weight INTEGER NOT NULL DEFAULT 0,
			target_metric TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatalf("seed legacy tables: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	requiredColumns := []string{
		"parent_kpi_id",
		"progress_q1",
		"progress_q2",
		"progress_q3",
		"progress_q4",
		"annual_progress",
	}
	for _, col := range requiredColumns {
		exists, err := columnExists(db, "kpis", col)
		if err != nil {
			t.Fatalf("columnExists(%s): %v", col, err)
		}
		if !exists {
			t.Fatalf("expected %s column to exist after migration", col)
		}
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_kpis_parent_id'`).Scan(&count); err != nil {
		t.Fatalf("check parent index: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idx_kpis_parent_id to exist, count=%d", count)
	}
}
