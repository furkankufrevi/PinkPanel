package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Load with no config file — should use defaults
	cfg, err := Load("/nonexistent/path/pinkpanel.yml")
	if err == nil {
		// If it doesn't error, check defaults
		if cfg.Server.Port != 8443 {
			t.Errorf("Expected default port 8443, got %d", cfg.Server.Port)
		}
	}

	// Load from environment
	t.Setenv("PINKPANEL_SERVER_PORT", "9999")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected env override port 9999, got %d", cfg.Server.Port)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pinkpanel.yml")

	content := []byte(`
server:
  host: 127.0.0.1
  port: 9443
database:
  path: /tmp/test.db
  wal_mode: true
  busy_timeout_ms: 3000
  max_open_conns: 10
logging:
  level: debug
  console: true
`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Expected host 127.0.0.1, got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9443 {
		t.Errorf("Expected port 9443, got %d", cfg.Server.Port)
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Expected db path /tmp/test.db, got %s", cfg.Database.Path)
	}
	if cfg.Database.MaxOpenConns != 10 {
		t.Errorf("Expected max_open_conns 10, got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("Expected log level debug, got %s", cfg.Logging.Level)
	}
}
