package storage

import (
	"database/sql"
	"fmt"
)

func Migrate(db *sql.DB) error {
	statements := []string{
		usersSQL,
		kpisSQL,
		achievementsSQL,
		reportsSQL,
		jobsSQL,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Run column migrations for existing databases
	if err := migrateKPIsTable(db); err != nil {
		return fmt.Errorf("migrate kpis table: %w", err)
	}

	return nil
}

// migrateKPIsTable adds new columns for hierarchy and progress tracking
func migrateKPIsTable(db *sql.DB) error {
	columns := []struct {
		name string
		def  string
	}{
		{"parent_kpi_id", "INTEGER DEFAULT NULL REFERENCES kpis(id) ON DELETE CASCADE"},
		{"progress_q1", "INTEGER NOT NULL DEFAULT 0"},
		{"progress_q2", "INTEGER NOT NULL DEFAULT 0"},
		{"progress_q3", "INTEGER NOT NULL DEFAULT 0"},
		{"progress_q4", "INTEGER NOT NULL DEFAULT 0"},
		{"annual_progress", "INTEGER NOT NULL DEFAULT 0"},
	}

	for _, col := range columns {
		if exists, err := columnExists(db, "kpis", col.name); err != nil {
			return err
		} else if !exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE kpis ADD COLUMN %s %s", col.name, col.def)); err != nil {
				return fmt.Errorf("add column %s: %w", col.name, err)
			}
		}
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_kpis_parent_id ON kpis(parent_kpi_id)`); err != nil {
		return fmt.Errorf("create parent index: %w", err)
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

const usersSQL = `
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

const kpisSQL = `
CREATE TABLE IF NOT EXISTS kpis (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  quarter TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  weight INTEGER NOT NULL DEFAULT 0,
  target_metric TEXT NOT NULL DEFAULT '',
  parent_kpi_id INTEGER DEFAULT NULL,
  progress_q1 INTEGER NOT NULL DEFAULT 0,
  progress_q2 INTEGER NOT NULL DEFAULT 0,
  progress_q3 INTEGER NOT NULL DEFAULT 0,
  progress_q4 INTEGER NOT NULL DEFAULT 0,
  annual_progress INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_kpi_id) REFERENCES kpis(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_kpis_user_quarter ON kpis(user_id, quarter);`

const achievementsSQL = `
CREATE TABLE IF NOT EXISTS achievements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  quarter TEXT NOT NULL,
  raw_text TEXT NOT NULL,
  enhanced_text TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  impact_note TEXT NOT NULL DEFAULT '',
  kpi_id INTEGER,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (kpi_id) REFERENCES kpis(id) ON DELETE SET NULL
);`

const reportsSQL = `
CREATE TABLE IF NOT EXISTS reports (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  quarter TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, quarter),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`

const jobsSQL = `
CREATE TABLE IF NOT EXISTS jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL,
  type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  payload TEXT NOT NULL,
  error TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);`
