package redirect

import (
	"database/sql"
	"fmt"
	"strings"
)

type Redirect struct {
	ID           int64  `json:"id"`
	DomainID     int64  `json:"domain_id"`
	SourcePath   string `json:"source_path"`
	TargetURL    string `json:"target_url"`
	RedirectType int    `json:"redirect_type"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) List(domainID int64) ([]Redirect, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, source_path, target_url, redirect_type, enabled, created_at, updated_at
		 FROM redirects WHERE domain_id = ? ORDER BY source_path`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirects []Redirect
	for rows.Next() {
		var r Redirect
		if err := rows.Scan(&r.ID, &r.DomainID, &r.SourcePath, &r.TargetURL, &r.RedirectType, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		redirects = append(redirects, r)
	}
	return redirects, rows.Err()
}

func (s *Service) GetByID(id int64) (*Redirect, error) {
	var r Redirect
	err := s.DB.QueryRow(
		`SELECT id, domain_id, source_path, target_url, redirect_type, enabled, created_at, updated_at
		 FROM redirects WHERE id = ?`, id,
	).Scan(&r.ID, &r.DomainID, &r.SourcePath, &r.TargetURL, &r.RedirectType, &r.Enabled, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("redirect not found")
	}
	return &r, nil
}

func (s *Service) Create(domainID int64, sourcePath, targetURL string, redirectType int) (*Redirect, error) {
	if err := validate(sourcePath, targetURL, redirectType); err != nil {
		return nil, err
	}

	// Check for duplicate source path
	var exists int
	s.DB.QueryRow(`SELECT COUNT(*) FROM redirects WHERE domain_id = ? AND source_path = ?`, domainID, sourcePath).Scan(&exists)
	if exists > 0 {
		return nil, fmt.Errorf("a redirect for this path already exists")
	}

	res, err := s.DB.Exec(
		`INSERT INTO redirects (domain_id, source_path, target_url, redirect_type) VALUES (?, ?, ?, ?)`,
		domainID, sourcePath, targetURL, redirectType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redirect: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) Update(id int64, sourcePath, targetURL *string, redirectType *int, enabled *bool) (*Redirect, error) {
	sets := []string{}
	args := []any{}

	if sourcePath != nil {
		if err := validatePath(*sourcePath); err != nil {
			return nil, err
		}
		sets = append(sets, "source_path = ?")
		args = append(args, *sourcePath)
	}
	if targetURL != nil {
		if strings.TrimSpace(*targetURL) == "" {
			return nil, fmt.Errorf("target URL is required")
		}
		sets = append(sets, "target_url = ?")
		args = append(args, *targetURL)
	}
	if redirectType != nil {
		if *redirectType != 301 && *redirectType != 302 {
			return nil, fmt.Errorf("redirect type must be 301 or 302")
		}
		sets = append(sets, "redirect_type = ?")
		args = append(args, *redirectType)
	}
	if enabled != nil {
		sets = append(sets, "enabled = ?")
		if *enabled {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	_, err := s.DB.Exec(
		fmt.Sprintf("UPDATE redirects SET %s WHERE id = ?", strings.Join(sets, ", ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *Service) Delete(id int64) (*Redirect, error) {
	r, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM redirects WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return r, nil
}

// ListEnabled returns all enabled redirects for a domain (for nginx config generation).
func (s *Service) ListEnabled(domainID int64) ([]Redirect, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, source_path, target_url, redirect_type, enabled, created_at, updated_at
		 FROM redirects WHERE domain_id = ? AND enabled = 1 ORDER BY source_path`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var redirects []Redirect
	for rows.Next() {
		var r Redirect
		if err := rows.Scan(&r.ID, &r.DomainID, &r.SourcePath, &r.TargetURL, &r.RedirectType, &r.Enabled, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		redirects = append(redirects, r)
	}
	return redirects, rows.Err()
}

func validate(sourcePath, targetURL string, redirectType int) error {
	if err := validatePath(sourcePath); err != nil {
		return err
	}
	if strings.TrimSpace(targetURL) == "" {
		return fmt.Errorf("target URL is required")
	}
	if redirectType != 301 && redirectType != 302 {
		return fmt.Errorf("redirect type must be 301 or 302")
	}
	return nil
}

func validatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("source path is required")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("source path must start with /")
	}
	return nil
}
