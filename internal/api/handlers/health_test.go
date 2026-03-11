package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestHealthEndpoint(t *testing.T) {
	db := setupTestDB(t)

	handler := &HealthHandler{
		DB:          db,
		AgentSocket: "",
		Version:     "test",
		StartTime:   time.Now(),
	}

	app := fiber.New()
	app.Get("/api/health", handler.Health)

	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result healthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Expected status ok, got %s", result.Status)
	}
	if result.Components.Database != "ok" {
		t.Errorf("Expected database ok, got %s", result.Components.Database)
	}
}

func TestHealthDetailedEndpoint(t *testing.T) {
	db := setupTestDB(t)

	handler := &HealthHandler{
		DB:          db,
		AgentSocket: "",
		Version:     "1.0.0-test",
		StartTime:   time.Now().Add(-5 * time.Minute),
	}

	app := fiber.New()
	app.Get("/api/health/detailed", handler.HealthDetailed)

	req := httptest.NewRequest("GET", "/api/health/detailed", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result detailedHealthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Version != "1.0.0-test" {
		t.Errorf("Expected version 1.0.0-test, got %s", result.Version)
	}
	if result.Uptime == "" {
		t.Error("Expected non-empty uptime")
	}
}
