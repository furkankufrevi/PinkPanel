package subdomain

import (
	"database/sql"
	"fmt"
	"regexp"
)

var subdomainNameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

type Subdomain struct {
	ID           int64  `json:"id"`
	DomainID     int64  `json:"domain_id"`
	Name         string `json:"name"`
	DocumentRoot string `json:"document_root"`
	CreatedAt    string `json:"created_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) List(domainID int64) ([]Subdomain, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, name, document_root, created_at FROM subdomains WHERE domain_id = ? ORDER BY name`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subdomain
	for rows.Next() {
		var sub Subdomain
		if err := rows.Scan(&sub.ID, &sub.DomainID, &sub.Name, &sub.DocumentRoot, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (s *Service) GetByID(id int64) (*Subdomain, error) {
	var sub Subdomain
	err := s.DB.QueryRow(
		`SELECT id, domain_id, name, document_root, created_at FROM subdomains WHERE id = ?`, id,
	).Scan(&sub.ID, &sub.DomainID, &sub.Name, &sub.DocumentRoot, &sub.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("subdomain not found")
	}
	return &sub, nil
}

func (s *Service) Create(domainID int64, name, domainName string) (*Subdomain, error) {
	if !subdomainNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid subdomain name: only alphanumeric and hyphens allowed")
	}
	if len(name) < 1 || len(name) > 63 {
		return nil, fmt.Errorf("subdomain name must be 1-63 characters")
	}

	documentRoot := fmt.Sprintf("/var/www/%s.%s/public", name, domainName)

	res, err := s.DB.Exec(
		`INSERT INTO subdomains (domain_id, name, document_root) VALUES (?, ?, ?)`,
		domainID, name, documentRoot,
	)
	if err != nil {
		return nil, fmt.Errorf("subdomain already exists or invalid domain")
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) Delete(id int64) (*Subdomain, error) {
	sub, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM subdomains WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) Count() (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM subdomains`).Scan(&count)
	return count, err
}
