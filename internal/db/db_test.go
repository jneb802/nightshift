package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	tables := []string{
		"schema_version",
		"projects",
		"task_history",
		"assigned_tasks",
		"run_history",
		"snapshots",
	}

	for _, table := range tables {
		if !tableExists(t, database.SQL(), table) {
			t.Fatalf("expected table %q to exist", table)
		}
	}
}

func TestOpenIdempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	database, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer database.Close()

	var count int
	row := database.SQL().QueryRow(`SELECT COUNT(*) FROM schema_version`)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_version count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 schema_version row, got %d", count)
	}
}

func TestMigrationVersioning(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	orig := make([]Migration, len(migrations))
	copy(orig, migrations)
	defer func() {
		migrations = orig
	}()

	dbPath := filepath.Join(t.TempDir(), "nightshift.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	migrations = append(migrations, Migration{
		Version:     2,
		Description: "add test table",
		SQL:         `CREATE TABLE migration_test (id INTEGER);`,
	})

	database, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer database.Close()

	version, err := CurrentVersion(database.SQL())
	if err != nil {
		t.Fatalf("current version: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected version 2, got %d", version)
	}

	if !tableExists(t, database.SQL(), "migration_test") {
		t.Fatalf("expected migration_test table to exist")
	}
}

func TestCurrentVersionFresh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dbPath := filepath.Join(t.TempDir(), "nightshift.db")

	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	defer sqlDB.Close()

	if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at DATETIME)`); err != nil {
		t.Fatalf("create schema_version: %v", err)
	}

	version, err := CurrentVersion(sqlDB)
	if err != nil {
		t.Fatalf("current version: %v", err)
	}
	if version != 0 {
		t.Fatalf("expected version 0, got %d", version)
	}
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()

	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name)
	var got string
	if err := row.Scan(&got); err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		t.Fatalf("query sqlite_master: %v", err)
	}
	return got == name
}
