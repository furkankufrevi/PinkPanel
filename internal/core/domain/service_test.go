package domain

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	db.Exec("PRAGMA foreign_keys=ON")
	db.Exec(`CREATE TABLE domains (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		document_root TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'active',
		php_version TEXT NOT NULL DEFAULT '8.3',
		parent_id INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE CASCADE,
		separate_dns INTEGER NOT NULL DEFAULT 0,
		modsecurity_enabled INTEGER NOT NULL DEFAULT 0,
		admin_id INTEGER DEFAULT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateRootDomain(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d, err := svc.Create("example.com", "8.3", nil, 1)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if d.Name != "example.com" {
		t.Errorf("Expected name example.com, got %s", d.Name)
	}
	if d.ParentID != nil {
		t.Errorf("Expected nil parent_id, got %v", d.ParentID)
	}
	if d.SeparateDNS {
		t.Error("Expected separate_dns false")
	}
	if d.DocumentRoot != "/var/www/example.com/phtml" {
		t.Errorf("Unexpected document_root: %s", d.DocumentRoot)
	}
	if d.Status != "active" {
		t.Errorf("Expected status active, got %s", d.Status)
	}
}

func TestCreateSubdomain(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, err := svc.Create("example.com", "8.3", nil, 1)
	if err != nil {
		t.Fatalf("Create parent failed: %v", err)
	}

	child, err := svc.Create("blog.example.com", "8.2", &parent.ID, 1)
	if err != nil {
		t.Fatalf("Create subdomain failed: %v", err)
	}

	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("Expected parent_id %d, got %v", parent.ID, child.ParentID)
	}
	if child.PHPVersion != "8.2" {
		t.Errorf("Expected php_version 8.2, got %s", child.PHPVersion)
	}
}

func TestPreventSubSubdomain(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	child, _ := svc.Create("blog.example.com", "8.3", &parent.ID, 1)

	_, err := svc.Create("test.blog.example.com", "8.3", &child.ID, 1)
	if err == nil {
		t.Error("Expected error when creating sub-subdomain")
	}
}

func TestCreateDuplicateDomain(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	_, err := svc.Create("example.com", "8.3", nil, 1)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = svc.Create("example.com", "8.3", nil, 1)
	if err == nil {
		t.Error("Expected error for duplicate domain")
	}
}

func TestCreateInvalidDomainName(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	_, err := svc.Create("not a domain", "8.3", nil, 1)
	if err == nil {
		t.Error("Expected error for invalid domain name")
	}
}

func TestCreateSubdomainParentNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	badID := int64(999)
	_, err := svc.Create("sub.example.com", "8.3", &badID, 1)
	if err == nil {
		t.Error("Expected error for nonexistent parent")
	}
}

func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	created, _ := svc.Create("example.com", "8.3", nil, 1)
	d, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if d.Name != "example.com" {
		t.Errorf("Expected example.com, got %s", d.Name)
	}
}

func TestGetByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	_, err := svc.GetByID(999)
	if err == nil {
		t.Error("Expected error for nonexistent ID")
	}
}

func TestGetByName(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	svc.Create("example.com", "8.3", nil, 1)
	d, err := svc.GetByName("example.com")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if d.Name != "example.com" {
		t.Errorf("Expected example.com, got %s", d.Name)
	}
}

func TestGetChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	svc.Create("blog.example.com", "8.3", &parent.ID, 1)
	svc.Create("api.example.com", "8.3", &parent.ID, 1)

	children, err := svc.GetChildren(parent.ID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(children))
	}
}

func TestGetChildrenEmpty(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	children, err := svc.GetChildren(parent.ID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(children))
	}
}

func TestListReturnsBothRootsAndChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	svc.Create("blog.example.com", "8.3", &parent.ID, 1)
	svc.Create("other.org", "8.3", nil, 1)

	domains, total, err := svc.List("", "", 1, 50, 0)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
	if len(domains) != 3 {
		t.Errorf("Expected 3 domains, got %d", len(domains))
	}
}

func TestListSearchFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	svc.Create("example.com", "8.3", nil, 1)
	svc.Create("other.org", "8.3", nil, 1)

	domains, total, err := svc.List("example", "", 1, 50, 0)
	if err != nil {
		t.Fatalf("List with search failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if len(domains) != 1 || domains[0].Name != "example.com" {
		t.Errorf("Expected example.com, got %v", domains)
	}
}

func TestListStatusFilter(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d1, _ := svc.Create("example.com", "8.3", nil, 1)
	svc.Create("other.org", "8.3", nil, 1)
	svc.Suspend(d1.ID)

	domains, total, err := svc.List("", "suspended", 1, 50, 0)
	if err != nil {
		t.Fatalf("List with status failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if len(domains) != 1 || domains[0].Name != "example.com" {
		t.Errorf("Expected suspended example.com, got %v", domains)
	}
}

func TestListPagination(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	for i := 0; i < 5; i++ {
		svc.Create(fmt.Sprintf("domain%d.com", i), "8.3", nil, 1)
	}

	domains, total, err := svc.List("", "", 1, 2, 0)
	if err != nil {
		t.Fatalf("List page 1 failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(domains) != 2 {
		t.Errorf("Expected 2 domains on page 1, got %d", len(domains))
	}
}

func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d, _ := svc.Create("example.com", "8.3", nil, 1)
	updated, err := svc.Update(d.ID, "/var/www/custom", "8.2")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.DocumentRoot != "/var/www/custom" {
		t.Errorf("Expected /var/www/custom, got %s", updated.DocumentRoot)
	}
	if updated.PHPVersion != "8.2" {
		t.Errorf("Expected 8.2, got %s", updated.PHPVersion)
	}
}

func TestUpdateSeparateDNS(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	child, _ := svc.Create("blog.example.com", "8.3", &parent.ID, 1)

	if child.SeparateDNS {
		t.Error("Expected separate_dns false initially")
	}

	updated, err := svc.UpdateSeparateDNS(child.ID, true)
	if err != nil {
		t.Fatalf("UpdateSeparateDNS failed: %v", err)
	}
	if !updated.SeparateDNS {
		t.Error("Expected separate_dns true after update")
	}

	// Toggle back
	updated, err = svc.UpdateSeparateDNS(child.ID, false)
	if err != nil {
		t.Fatalf("UpdateSeparateDNS(false) failed: %v", err)
	}
	if updated.SeparateDNS {
		t.Error("Expected separate_dns false after toggle back")
	}
}

func TestSuspendAndActivate(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d, _ := svc.Create("example.com", "8.3", nil, 1)

	if err := svc.Suspend(d.ID); err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	suspended, _ := svc.GetByID(d.ID)
	if suspended.Status != "suspended" {
		t.Errorf("Expected suspended, got %s", suspended.Status)
	}

	if err := svc.Activate(d.ID); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	activated, _ := svc.GetByID(d.ID)
	if activated.Status != "active" {
		t.Errorf("Expected active, got %s", activated.Status)
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d, _ := svc.Create("example.com", "8.3", nil, 1)
	if err := svc.Delete(d.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := svc.GetByID(d.ID)
	if err == nil {
		t.Error("Expected error getting deleted domain")
	}
}

func TestDeleteCascadesToChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	parent, _ := svc.Create("example.com", "8.3", nil, 1)
	child, _ := svc.Create("blog.example.com", "8.3", &parent.ID, 1)

	if err := svc.Delete(parent.ID); err != nil {
		t.Fatalf("Delete parent failed: %v", err)
	}

	_, err := svc.GetByID(child.ID)
	if err == nil {
		t.Error("Expected child to be cascade-deleted")
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	count, err := svc.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}

	svc.Create("example.com", "8.3", nil, 1)
	count, _ = svc.Count()
	if count != 1 {
		t.Errorf("Expected 1, got %d", count)
	}
}

func TestBuildDocumentRoot(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		domain, sysUser, expected string
	}{
		{"example.com", "", "/var/www/example.com/phtml"},
		{"example.com", "www-data", "/var/www/example.com/phtml"},
		{"example.com", "pp_john", "/home/pp_john/domains/example.com/public"},
	}

	for _, tc := range tests {
		got := svc.BuildDocumentRoot(tc.domain, tc.sysUser)
		if got != tc.expected {
			t.Errorf("BuildDocumentRoot(%q, %q) = %q, want %q", tc.domain, tc.sysUser, got, tc.expected)
		}
	}
}

func TestCreateForUser(t *testing.T) {
	db := setupTestDB(t)
	svc := &Service{DB: db}

	d, err := svc.CreateForUser("example.com", "8.3", nil, 1, "pp_john")
	if err != nil {
		t.Fatalf("CreateForUser failed: %v", err)
	}

	if d.DocumentRoot != "/home/pp_john/domains/example.com/public" {
		t.Errorf("Expected user-scoped doc root, got %s", d.DocumentRoot)
	}
}
