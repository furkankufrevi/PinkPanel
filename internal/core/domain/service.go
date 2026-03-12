// Package domain provides the domain management service layer.
package domain

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// domainNameRe validates domain names: letters, numbers, hyphens, dots,
// at least two parts, TLD between 2 and 63 characters.
var domainNameRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}$`)

// Domain represents a row in the domains table.
type Domain struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	DocumentRoot string `json:"document_root"`
	Status       string `json:"status"`
	PHPVersion   string `json:"php_version"`
	ParentID     *int64 `json:"parent_id"`
	SeparateDNS  bool   `json:"separate_dns"`
	AdminID      *int64 `json:"admin_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Service provides domain-related database operations.
type Service struct {
	DB *sql.DB
}

const domainColumns = "id, name, document_root, status, php_version, parent_id, separate_dns, admin_id, created_at, updated_at"

func scanDomain(row interface{ Scan(...interface{}) error }) (*Domain, error) {
	d := &Domain{}
	var separateDNS int
	err := row.Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.Status, &d.PHPVersion, &d.ParentID, &separateDNS, &d.AdminID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	d.SeparateDNS = separateDNS != 0
	return d, nil
}

// List returns a paginated slice of domains with optional search, status,
// and admin_id filtering. It also returns the total count of matching rows.
// Pass adminID=0 to skip ownership filtering (super_admin sees all).
func (s *Service) List(search string, status string, page int, perPage int, adminID int64) ([]Domain, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if adminID > 0 {
		where = append(where, "admin_id = ?")
		args = append(args, adminID)
	}
	if search != "" {
		where = append(where, "name LIKE ?")
		args = append(args, "%"+search+"%")
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}

	clause := strings.Join(where, " AND ")

	// Total count.
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM domains WHERE %s", clause)
	if err := s.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting domains: %w", err)
	}

	// Paginated rows.
	offset := (page - 1) * perPage
	listQuery := fmt.Sprintf(
		"SELECT %s FROM domains WHERE %s ORDER BY parent_id NULLS FIRST, id DESC LIMIT ? OFFSET ?",
		domainColumns, clause,
	)
	listArgs := append(args, perPage, offset)

	rows, err := s.DB.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing domains: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		d, err := scanDomain(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning domain row: %w", err)
		}
		domains = append(domains, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating domain rows: %w", err)
	}

	return domains, total, nil
}

// GetByID returns a single domain by its primary key.
func (s *Service) GetByID(id int64) (*Domain, error) {
	row := s.DB.QueryRow(
		fmt.Sprintf("SELECT %s FROM domains WHERE id = ?", domainColumns), id,
	)
	d, err := scanDomain(row)
	if err != nil {
		return nil, fmt.Errorf("getting domain by id: %w", err)
	}
	return d, nil
}

// GetByName returns a single domain by its unique name.
func (s *Service) GetByName(name string) (*Domain, error) {
	row := s.DB.QueryRow(
		fmt.Sprintf("SELECT %s FROM domains WHERE name = ?", domainColumns), name,
	)
	d, err := scanDomain(row)
	if err != nil {
		return nil, fmt.Errorf("getting domain by name: %w", err)
	}
	return d, nil
}

// GetChildren returns all subdomains (domains with parent_id = parentID).
func (s *Service) GetChildren(parentID int64) ([]Domain, error) {
	rows, err := s.DB.Query(
		fmt.Sprintf("SELECT %s FROM domains WHERE parent_id = ? ORDER BY name", domainColumns), parentID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing child domains: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		d, err := scanDomain(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning child domain row: %w", err)
		}
		domains = append(domains, *d)
	}
	return domains, rows.Err()
}

// BuildDocumentRoot returns the document root path for a domain.
// If systemUsername is empty or "www-data", uses /var/www/{domain}/phtml.
// Otherwise uses /home/{systemUsername}/domains/{domain}/public.
func (s *Service) BuildDocumentRoot(domainName, systemUsername string) string {
	if systemUsername == "" || systemUsername == "www-data" {
		return fmt.Sprintf("/var/www/%s/phtml", domainName)
	}
	return fmt.Sprintf("/home/%s/domains/%s/public", systemUsername, domainName)
}

// CreateForUser creates a domain with a user-scoped document root.
func (s *Service) CreateForUser(name, phpVersion string, parentID *int64, adminID int64, systemUsername string) (*Domain, error) {
	if !domainNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid domain name: %s", name)
	}

	if parentID != nil {
		parent, err := s.GetByID(*parentID)
		if err != nil {
			return nil, fmt.Errorf("parent domain not found")
		}
		if parent.ParentID != nil {
			return nil, fmt.Errorf("cannot create subdomain of a subdomain")
		}
	}

	var exists int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking duplicate domain: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("domain already exists: %s", name)
	}

	documentRoot := s.BuildDocumentRoot(name, systemUsername)

	result, err := s.DB.Exec(
		"INSERT INTO domains (name, document_root, php_version, parent_id, separate_dns, admin_id) VALUES (?, ?, ?, ?, 0, ?)",
		name, documentRoot, phpVersion, parentID, adminID,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting domain: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Create validates the domain name, checks for duplicates, and inserts a new
// domain. If parentID is non-nil, the domain is treated as a subdomain.
func (s *Service) Create(name string, phpVersion string, parentID *int64, adminID int64) (*Domain, error) {
	if !domainNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid domain name: %s", name)
	}

	// Prevent sub-subdomains
	if parentID != nil {
		parent, err := s.GetByID(*parentID)
		if err != nil {
			return nil, fmt.Errorf("parent domain not found")
		}
		if parent.ParentID != nil {
			return nil, fmt.Errorf("cannot create subdomain of a subdomain")
		}
	}

	// Check for duplicate.
	var exists int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking duplicate domain: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("domain already exists: %s", name)
	}

	documentRoot := s.BuildDocumentRoot(name, "")

	result, err := s.DB.Exec(
		"INSERT INTO domains (name, document_root, php_version, parent_id, separate_dns, admin_id) VALUES (?, ?, ?, ?, 0, ?)",
		name, documentRoot, phpVersion, parentID, adminID,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting domain: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update modifies a domain's document_root and php_version.
func (s *Service) Update(id int64, documentRoot string, phpVersion string) (*Domain, error) {
	_, err := s.DB.Exec(
		"UPDATE domains SET document_root = ?, php_version = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		documentRoot, phpVersion, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating domain: %w", err)
	}
	return s.GetByID(id)
}

// UpdateSeparateDNS toggles the separate_dns flag for a subdomain.
func (s *Service) UpdateSeparateDNS(id int64, separateDNS bool) (*Domain, error) {
	val := 0
	if separateDNS {
		val = 1
	}
	_, err := s.DB.Exec(
		"UPDATE domains SET separate_dns = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		val, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating separate_dns: %w", err)
	}
	return s.GetByID(id)
}

// Suspend sets a domain's status to 'suspended'.
func (s *Service) Suspend(id int64) error {
	_, err := s.DB.Exec(
		"UPDATE domains SET status = 'suspended', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id,
	)
	if err != nil {
		return fmt.Errorf("suspending domain: %w", err)
	}
	return nil
}

// Activate sets a domain's status to 'active'.
func (s *Service) Activate(id int64) error {
	_, err := s.DB.Exec(
		"UPDATE domains SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id,
	)
	if err != nil {
		return fmt.Errorf("activating domain: %w", err)
	}
	return nil
}

// Delete removes a domain from the database.
func (s *Service) Delete(id int64) error {
	_, err := s.DB.Exec("DELETE FROM domains WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting domain: %w", err)
	}
	return nil
}

// Count returns the total number of domains.
func (s *Service) Count() (int, error) {
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM domains").Scan(&count); err != nil {
		return 0, fmt.Errorf("counting domains: %w", err)
	}
	return count, nil
}
