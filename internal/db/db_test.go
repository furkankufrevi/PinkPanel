package db

import (
	"path/filepath"
	"testing"

	"github.com/pinkpanel/pinkpanel/internal/config"
)

func TestOpenAndMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	cfg := config.DatabaseConfig{
		Path:         dbPath,
		WALMode:      true,
		BusyTimeout:  5000,
		MaxOpenConns: 5,
	}

	database, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Test ping
	if err := database.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Run migrations
	if err := Migrate(database); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify settings table exists and has data
	var value string
	err = database.QueryRow("SELECT value FROM settings WHERE key = ?", "panel.name").Scan(&value)
	if err != nil {
		t.Fatalf("Failed to query settings: %v", err)
	}
	if value != "PinkPanel" {
		t.Errorf("Expected panel.name=PinkPanel, got %s", value)
	}

	// Verify WAL mode
	var journalMode string
	err = database.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected WAL journal mode, got %s", journalMode)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	cfg := config.DatabaseConfig{
		Path:         dbPath,
		WALMode:      true,
		BusyTimeout:  5000,
		MaxOpenConns: 5,
	}

	database, err := Open(cfg)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations twice — should not error
	if err := Migrate(database); err != nil {
		t.Fatalf("First migration failed: %v", err)
	}
	if err := Migrate(database); err != nil {
		t.Fatalf("Second migration failed (not idempotent): %v", err)
	}
}
