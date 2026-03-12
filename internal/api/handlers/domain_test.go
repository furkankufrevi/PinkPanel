package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	_ "modernc.org/sqlite"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
)

func setupDomainTestDB(t *testing.T) (*DomainHandler, *fiber.App) {
	t.Helper()
	db := setupTestDB(t)

	// Create required tables
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec(`CREATE TABLE domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		document_root TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		php_version TEXT NOT NULL DEFAULT '8.3',
		parent_id INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE CASCADE,
		separate_dns INTEGER NOT NULL DEFAULT 0,
		admin_id INTEGER DEFAULT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE dns_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain_id INTEGER NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		ttl INTEGER NOT NULL DEFAULT 3600,
		priority INTEGER,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	db.Exec(`CREATE TABLE activity_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_id INTEGER,
		action TEXT NOT NULL,
		target_type TEXT,
		target_id INTEGER,
		details TEXT,
		ip_address TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	domainSvc := &domain.Service{DB: db}
	dnsSvc := &dns.Service{DB: db}
	agentClient := agent.NewClient("/tmp/nonexistent.sock")

	handler := &DomainHandler{
		DB:          db,
		DomainSvc:   domainSvc,
		DNSSvc:      dnsSvc,
		AgentClient: agentClient,
	}

	app := fiber.New()
	// Inject test admin context (super_admin with id=1)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("admin_id", int64(1))
		c.Locals("username", "admin")
		c.Locals("role", "super_admin")
		return c.Next()
	})
	app.Get("/api/domains", handler.List)
	app.Post("/api/domains", handler.Create)
	app.Get("/api/domains/:id", handler.Get)
	app.Put("/api/domains/:id", handler.Update)
	app.Delete("/api/domains/:id", handler.Delete)
	app.Post("/api/domains/:id/suspend", handler.Suspend)
	app.Post("/api/domains/:id/activate", handler.Activate)

	return handler, app
}

func parseDomainResponse(respBody []byte) domain.Domain {
	// Response may be wrapped {"data":..., "warnings":...} or bare domain
	var wrapped struct {
		Data domain.Domain `json:"data"`
	}
	json.Unmarshal(respBody, &wrapped)
	if wrapped.Data.Name != "" {
		return wrapped.Data
	}
	var d domain.Domain
	json.Unmarshal(respBody, &d)
	return d
}

func TestDomainHandlerCreateRootDomain(t *testing.T) {
	_, app := setupDomainTestDB(t)

	body, _ := json.Marshal(createDomainRequest{
		Name:       "example.com",
		PHPVersion: "8.3",
	})
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, _ := io.ReadAll(resp.Body)
	d := parseDomainResponse(respBody)

	if d.Name != "example.com" {
		t.Errorf("Expected name example.com, got %s", d.Name)
	}
	if d.ParentID != nil {
		t.Errorf("Expected nil parent_id, got %v", d.ParentID)
	}
}

func TestDomainHandlerCreateSubdomain(t *testing.T) {
	_, app := setupDomainTestDB(t)

	// Create parent
	body, _ := json.Marshal(createDomainRequest{Name: "example.com", PHPVersion: "8.3"})
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	respBody, _ := io.ReadAll(resp.Body)
	parent := parseDomainResponse(respBody)

	// Create subdomain with parent_id
	subBody := fmt.Sprintf(`{"name":"blog","php_version":"8.2","parent_id":%d}`, parent.ID)
	req = httptest.NewRequest("POST", "/api/domains", bytes.NewReader([]byte(subBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, _ = io.ReadAll(resp.Body)
	child := parseDomainResponse(respBody)

	if child.Name != "blog.example.com" {
		t.Errorf("Expected name blog.example.com, got %s", child.Name)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("Expected parent_id %d, got %v", parent.ID, child.ParentID)
	}
	if child.PHPVersion != "8.2" {
		t.Errorf("Expected PHP 8.2, got %s", child.PHPVersion)
	}
}

func TestDomainHandlerCreateSubSubdomainRejected(t *testing.T) {
	_, app := setupDomainTestDB(t)

	// Create parent
	body, _ := json.Marshal(createDomainRequest{Name: "example.com", PHPVersion: "8.3"})
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	respBody, _ := io.ReadAll(resp.Body)
	parent := parseDomainResponse(respBody)

	// Create subdomain
	subBody := fmt.Sprintf(`{"name":"blog","php_version":"8.3","parent_id":%d}`, parent.ID)
	req = httptest.NewRequest("POST", "/api/domains", bytes.NewReader([]byte(subBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)
	respBody, _ = io.ReadAll(resp.Body)
	child := parseDomainResponse(respBody)

	// Try sub-subdomain
	subSubBody := fmt.Sprintf(`{"name":"test","php_version":"8.3","parent_id":%d}`, child.ID)
	req = httptest.NewRequest("POST", "/api/domains", bytes.NewReader([]byte(subSubBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for sub-subdomain, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerCreateEmptyName(t *testing.T) {
	_, app := setupDomainTestDB(t)

	body, _ := json.Marshal(createDomainRequest{Name: "", PHPVersion: "8.3"})
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for empty name, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerCreateBadParentID(t *testing.T) {
	_, app := setupDomainTestDB(t)

	body := `{"name":"sub","php_version":"8.3","parent_id":999}`
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != 400 {
		t.Errorf("Expected 400 for bad parent_id, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerList(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	handler.DomainSvc.Create("example.com", "8.3", nil, 1)
	parent, _ := handler.DomainSvc.Create("other.org", "8.3", nil, 1)
	handler.DomainSvc.Create("blog.other.org", "8.3", &parent.ID, 1)

	req := httptest.NewRequest("GET", "/api/domains?per_page=50", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data    []domain.Domain `json:"data"`
		Total   int             `json:"total"`
		Page    int             `json:"page"`
		PerPage int             `json:"per_page"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	if result.Total != 3 {
		t.Errorf("Expected total 3, got %d", result.Total)
	}
	if len(result.Data) != 3 {
		t.Errorf("Expected 3 domains, got %d", len(result.Data))
	}

	// Verify parent_id is populated correctly
	var foundChild bool
	for _, d := range result.Data {
		if d.Name == "blog.other.org" {
			foundChild = true
			if d.ParentID == nil {
				t.Error("Expected blog.other.org to have parent_id set")
			}
		}
	}
	if !foundChild {
		t.Error("Expected to find blog.other.org in list")
	}
}

func TestDomainHandlerListSearch(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	handler.DomainSvc.Create("example.com", "8.3", nil, 1)
	handler.DomainSvc.Create("other.org", "8.3", nil, 1)

	req := httptest.NewRequest("GET", "/api/domains?search=example", nil)
	resp, _ := app.Test(req, -1)

	var result struct {
		Data  []domain.Domain `json:"data"`
		Total int             `json:"total"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)

	if result.Total != 1 {
		t.Errorf("Expected 1, got %d", result.Total)
	}
}

func TestDomainHandlerGet(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	d, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)

	req := httptest.NewRequest("GET", fmt.Sprintf("/api/domains/%d", d.ID), nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var got domain.Domain
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &got)

	if got.Name != "example.com" {
		t.Errorf("Expected example.com, got %s", got.Name)
	}
	if got.SeparateDNS != false {
		t.Error("Expected separate_dns false")
	}
}

func TestDomainHandlerGetNotFound(t *testing.T) {
	_, app := setupDomainTestDB(t)

	req := httptest.NewRequest("GET", "/api/domains/999", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerGetInvalidID(t *testing.T) {
	_, app := setupDomainTestDB(t)

	req := httptest.NewRequest("GET", "/api/domains/abc", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 400 {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerUpdate(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	d, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)

	body, _ := json.Marshal(updateDomainRequest{
		DocumentRoot: "/var/www/custom",
		PHPVersion:   "8.2",
	})
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/domains/%d", d.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var updated domain.Domain
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &updated)

	if updated.DocumentRoot != "/var/www/custom" {
		t.Errorf("Expected /var/www/custom, got %s", updated.DocumentRoot)
	}
	if updated.PHPVersion != "8.2" {
		t.Errorf("Expected 8.2, got %s", updated.PHPVersion)
	}
}

func TestDomainHandlerUpdateSeparateDNS(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	parent, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)
	child, _ := handler.DomainSvc.Create("blog.example.com", "8.3", &parent.ID, 1)

	// Toggle separate_dns on
	body := `{"separate_dns":true}`
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/domains/%d", child.ID), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var updated domain.Domain
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &updated)

	if !updated.SeparateDNS {
		t.Error("Expected separate_dns true after toggle")
	}
}

func TestDomainHandlerUpdateNotFound(t *testing.T) {
	_, app := setupDomainTestDB(t)

	body, _ := json.Marshal(updateDomainRequest{PHPVersion: "8.2"})
	req := httptest.NewRequest("PUT", "/api/domains/999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerSuspendAndActivate(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	d, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)

	// Suspend
	req := httptest.NewRequest("POST", fmt.Sprintf("/api/domains/%d/suspend", d.ID), nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	var suspended domain.Domain
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &suspended)
	if suspended.Status != "suspended" {
		t.Errorf("Expected suspended, got %s", suspended.Status)
	}

	// Activate
	req = httptest.NewRequest("POST", fmt.Sprintf("/api/domains/%d/activate", d.ID), nil)
	resp, _ = app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	var activated domain.Domain
	respBody, _ = io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &activated)
	if activated.Status != "active" {
		t.Errorf("Expected active, got %s", activated.Status)
	}
}

func TestDomainHandlerSuspendNotFound(t *testing.T) {
	_, app := setupDomainTestDB(t)

	req := httptest.NewRequest("POST", "/api/domains/999/suspend", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerDelete(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	d, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/domains/%d", d.ID), nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Verify deleted
	_, err := handler.DomainSvc.GetByID(d.ID)
	if err == nil {
		t.Error("Expected domain to be deleted")
	}
}

func TestDomainHandlerDeleteNotFound(t *testing.T) {
	_, app := setupDomainTestDB(t)

	req := httptest.NewRequest("DELETE", "/api/domains/999", nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 404 {
		t.Errorf("Expected 404, got %d", resp.StatusCode)
	}
}

func TestDomainHandlerDeleteCascadesChildren(t *testing.T) {
	handler, app := setupDomainTestDB(t)

	parent, _ := handler.DomainSvc.Create("example.com", "8.3", nil, 1)
	child, _ := handler.DomainSvc.Create("blog.example.com", "8.3", &parent.ID, 1)

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/domains/%d", parent.ID), nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	_, err := handler.DomainSvc.GetByID(child.ID)
	if err == nil {
		t.Error("Expected child to be cascade-deleted")
	}
}

func TestDomainHandlerDefaultPHPVersion(t *testing.T) {
	_, app := setupDomainTestDB(t)

	// Create without specifying php_version
	body := []byte(`{"name":"example.com"}`)
	req := httptest.NewRequest("POST", "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, _ := io.ReadAll(resp.Body)
	d := parseDomainResponse(respBody)

	if d.PHPVersion != "8.3" {
		t.Errorf("Expected default PHP 8.3, got %s", d.PHPVersion)
	}
}

func TestExtractSubPrefix(t *testing.T) {
	tests := []struct {
		fqdn, parent, want string
	}{
		{"blog.example.com", "example.com", "blog"},
		{"api.example.com", "example.com", "api"},
		{"deep.sub.example.com", "example.com", "deep.sub"},
	}
	for _, tc := range tests {
		got := extractSubPrefix(tc.fqdn, tc.parent)
		if got != tc.want {
			t.Errorf("extractSubPrefix(%q, %q) = %q, want %q", tc.fqdn, tc.parent, got, tc.want)
		}
	}
}

func TestContainsSuffix(t *testing.T) {
	tests := []struct {
		name, suffix string
		want         bool
	}{
		{"blog.example.com", "example.com", true},
		{"example.com", "example.com", false},
		{"other.org", "example.com", false},
	}
	for _, tc := range tests {
		got := containsSuffix(tc.name, tc.suffix)
		if got != tc.want {
			t.Errorf("containsSuffix(%q, %q) = %v, want %v", tc.name, tc.suffix, got, tc.want)
		}
	}
}
