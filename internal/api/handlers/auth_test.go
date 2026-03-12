package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/pinkpanel/pinkpanel/internal/auth"
)

func setupAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}

	// Create tables
	db.Exec(`CREATE TABLE admins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'super_admin',
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE refresh_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_id INTEGER NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE login_attempts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		success INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`INSERT INTO settings (key, value) VALUES ('panel.setup_complete', 'false')`)

	t.Cleanup(func() { db.Close() })
	return db
}

func TestLoginFlow(t *testing.T) {
	db := setupAuthTestDB(t)
	jwtManager, _ := auth.NewJWTManager("", 15*time.Minute, 7*24*time.Hour)

	// Create a test admin
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), 4)
	db.Exec("INSERT INTO admins (username, email, password_hash) VALUES (?, ?, ?)",
		"admin", "admin@test.com", string(hash))

	handler := &AuthHandler{
		DB:         db,
		JWTManager: jwtManager,
		BcryptCost: 4,
	}

	app := fiber.New()
	app.Post("/api/auth/login", handler.Login)

	// Test successful login
	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "testpass123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenPair auth.TokenPair
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &tokenPair); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if tokenPair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}
	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}

	// Test wrong password
	body, _ = json.Marshal(loginRequest{Username: "admin", Password: "wrongpass"})
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 for wrong password, got %d", resp.StatusCode)
	}

	// Test nonexistent user
	body, _ = json.Marshal(loginRequest{Username: "nouser", Password: "testpass123"})
	req = httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 for nonexistent user, got %d", resp.StatusCode)
	}
}

func TestRefreshFlow(t *testing.T) {
	db := setupAuthTestDB(t)
	jwtManager, _ := auth.NewJWTManager("", 15*time.Minute, 7*24*time.Hour)

	// Create admin and get initial tokens via login
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), 4)
	db.Exec("INSERT INTO admins (username, email, password_hash) VALUES (?, ?, ?)",
		"admin", "admin@test.com", string(hash))

	authHandler := &AuthHandler{
		DB:         db,
		JWTManager: jwtManager,
		BcryptCost: 4,
	}

	app := fiber.New()
	app.Post("/api/auth/login", authHandler.Login)
	app.Post("/api/auth/refresh", authHandler.Refresh)

	// Login first
	body, _ := json.Marshal(loginRequest{Username: "admin", Password: "testpass123"})
	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	var tokenPair auth.TokenPair
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &tokenPair)

	// Refresh
	body, _ = json.Marshal(refreshRequest{RefreshToken: tokenPair.RefreshToken})
	req = httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var newPair auth.TokenPair
	respBody, _ = io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &newPair)
	if newPair.AccessToken == "" {
		t.Error("New access token should not be empty")
	}

	// Old refresh token should be invalidated (rotation)
	body, _ = json.Marshal(refreshRequest{RefreshToken: tokenPair.RefreshToken})
	req = httptest.NewRequest("POST", "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 401 {
		t.Errorf("Expected 401 for reused refresh token, got %d", resp.StatusCode)
	}
}

func TestSetupFlow(t *testing.T) {
	db := setupAuthTestDB(t)
	jwtManager, _ := auth.NewJWTManager("", 15*time.Minute, 7*24*time.Hour)

	handler := &SetupHandler{
		DB:         db,
		JWTManager: jwtManager,
		BcryptCost: 4,
	}

	app := fiber.New()
	app.Get("/api/setup/status", handler.Status)
	app.Post("/api/setup/admin", handler.CreateAdmin)

	// Check setup required
	req := httptest.NewRequest("GET", "/api/setup/status", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	var statusResp map[string]interface{}
	json.Unmarshal(respBody, &statusResp)
	if statusResp["setup_required"] != true {
		t.Error("Setup should be required when no admins exist")
	}

	// Create admin
	body, _ := json.Marshal(setupRequest{
		Username: "admin",
		Email:    "admin@test.com",
		Password: "securepass123",
	})
	req = httptest.NewRequest("POST", "/api/setup/admin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Setup should no longer be required
	req = httptest.NewRequest("GET", "/api/setup/status", nil)
	resp, _ = app.Test(req)
	respBody, _ = io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &statusResp)
	if statusResp["setup_required"] != false {
		t.Error("Setup should not be required after admin creation")
	}

	// Cannot create another admin via setup
	body, _ = json.Marshal(setupRequest{
		Username: "admin2",
		Email:    "admin2@test.com",
		Password: "securepass123",
	})
	req = httptest.NewRequest("POST", "/api/setup/admin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != 409 {
		t.Errorf("Expected 409 for duplicate setup, got %d", resp.StatusCode)
	}
}
