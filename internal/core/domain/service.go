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
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Service provides domain-related database operations.
type Service struct {
	DB *sql.DB
}

// List returns a paginated slice of domains with optional search and status
// filtering. It also returns the total count of matching rows.
func (s *Service) List(search string, status string, page int, perPage int) ([]Domain, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

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
		"SELECT id, name, document_root, status, php_version, created_at, updated_at FROM domains WHERE %s ORDER BY id DESC LIMIT ? OFFSET ?",
		clause,
	)
	listArgs := append(args, perPage, offset)

	rows, err := s.DB.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing domains: %w", err)
	}
	defer rows.Close()

	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.Status, &d.PHPVersion, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning domain row: %w", err)
		}
		domains = append(domains, d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating domain rows: %w", err)
	}

	return domains, total, nil
}

// GetByID returns a single domain by its primary key.
func (s *Service) GetByID(id int64) (*Domain, error) {
	d := &Domain{}
	err := s.DB.QueryRow(
		"SELECT id, name, document_root, status, php_version, created_at, updated_at FROM domains WHERE id = ?", id,
	).Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.Status, &d.PHPVersion, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting domain by id: %w", err)
	}
	return d, nil
}

// GetByName returns a single domain by its unique name.
func (s *Service) GetByName(name string) (*Domain, error) {
	d := &Domain{}
	err := s.DB.QueryRow(
		"SELECT id, name, document_root, status, php_version, created_at, updated_at FROM domains WHERE name = ?", name,
	).Scan(&d.ID, &d.Name, &d.DocumentRoot, &d.Status, &d.PHPVersion, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting domain by name: %w", err)
	}
	return d, nil
}

// Create validates the domain name, checks for duplicates, and inserts a new
// domain with document_root set to /var/www/{name}/public.
func (s *Service) Create(name string, phpVersion string) (*Domain, error) {
	if !domainNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid domain name: %s", name)
	}

	// Check for duplicate.
	var exists int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM domains WHERE name = ?", name).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking duplicate domain: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("domain already exists: %s", name)
	}

	documentRoot := fmt.Sprintf("/var/www/%s/public", name)

	result, err := s.DB.Exec(
		"INSERT INTO domains (name, document_root, php_version) VALUES (?, ?, ?)",
		name, documentRoot, phpVersion,
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
