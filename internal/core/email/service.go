package email

import (
	"database/sql"
	"fmt"
	"regexp"
)

var safeLocalPartRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
var safeEmailRe = regexp.MustCompile(`^[a-zA-Z0-9._+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Account struct {
	ID        int64  `json:"id"`
	DomainID  int64  `json:"domain_id"`
	Address   string `json:"address"`
	QuotaMB   int64  `json:"quota_mb"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Forwarder struct {
	ID             int64  `json:"id"`
	DomainID       int64  `json:"domain_id"`
	SourceAddress  string `json:"source_address"`
	Destination    string `json:"destination"`
	CreatedAt      string `json:"created_at"`
}

type Service struct {
	DB *sql.DB
}

// ---------- Accounts ----------

func (s *Service) ListAccounts(domainID int64) ([]Account, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, address, quota_mb, enabled, created_at, updated_at
		 FROM email_accounts WHERE domain_id = ? ORDER BY address`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var enabled int
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Address, &a.QuotaMB, &enabled, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Enabled = enabled == 1
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *Service) GetAccountByID(id int64) (*Account, error) {
	var a Account
	var enabled int
	err := s.DB.QueryRow(
		`SELECT id, domain_id, address, quota_mb, enabled, created_at, updated_at
		 FROM email_accounts WHERE id = ?`, id,
	).Scan(&a.ID, &a.DomainID, &a.Address, &a.QuotaMB, &enabled, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("email account not found")
	}
	a.Enabled = enabled == 1
	return &a, nil
}

func (s *Service) CreateAccount(domainID int64, address string, quotaMB int64) (*Account, error) {
	if !safeLocalPartRe.MatchString(address) {
		return nil, fmt.Errorf("invalid email address: only alphanumeric, dots, hyphens, and underscores allowed")
	}
	if len(address) < 1 || len(address) > 64 {
		return nil, fmt.Errorf("address must be 1-64 characters")
	}

	res, err := s.DB.Exec(
		`INSERT INTO email_accounts (domain_id, address, quota_mb) VALUES (?, ?, ?)`,
		domainID, address, quotaMB,
	)
	if err != nil {
		return nil, fmt.Errorf("email address already exists or invalid domain")
	}
	id, _ := res.LastInsertId()
	return s.GetAccountByID(id)
}

func (s *Service) DeleteAccount(id int64) (*Account, error) {
	a, err := s.GetAccountByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM email_accounts WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) UpdateQuota(id int64, quotaMB int64) error {
	_, err := s.DB.Exec(`UPDATE email_accounts SET quota_mb = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, quotaMB, id)
	return err
}

func (s *Service) ToggleAccount(id int64, enabled bool) error {
	val := 0
	if enabled {
		val = 1
	}
	_, err := s.DB.Exec(`UPDATE email_accounts SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, val, id)
	return err
}

func (s *Service) CountAccounts() (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM email_accounts`).Scan(&count)
	return count, err
}

// ---------- Forwarders ----------

func (s *Service) ListForwarders(domainID int64) ([]Forwarder, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, source_address, destination, created_at
		 FROM email_forwarders WHERE domain_id = ? ORDER BY source_address`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var forwarders []Forwarder
	for rows.Next() {
		var f Forwarder
		if err := rows.Scan(&f.ID, &f.DomainID, &f.SourceAddress, &f.Destination, &f.CreatedAt); err != nil {
			return nil, err
		}
		forwarders = append(forwarders, f)
	}
	return forwarders, rows.Err()
}

func (s *Service) CreateForwarder(domainID int64, source, destination string) (*Forwarder, error) {
	if !safeLocalPartRe.MatchString(source) {
		return nil, fmt.Errorf("invalid source address")
	}
	if !safeEmailRe.MatchString(destination) {
		return nil, fmt.Errorf("invalid destination email address")
	}

	res, err := s.DB.Exec(
		`INSERT INTO email_forwarders (domain_id, source_address, destination) VALUES (?, ?, ?)`,
		domainID, source, destination,
	)
	if err != nil {
		return nil, fmt.Errorf("forwarder already exists")
	}
	id, _ := res.LastInsertId()

	var f Forwarder
	s.DB.QueryRow(
		`SELECT id, domain_id, source_address, destination, created_at FROM email_forwarders WHERE id = ?`, id,
	).Scan(&f.ID, &f.DomainID, &f.SourceAddress, &f.Destination, &f.CreatedAt)
	return &f, nil
}

func (s *Service) DeleteForwarder(id int64) (*Forwarder, error) {
	var f Forwarder
	err := s.DB.QueryRow(
		`SELECT id, domain_id, source_address, destination, created_at FROM email_forwarders WHERE id = ?`, id,
	).Scan(&f.ID, &f.DomainID, &f.SourceAddress, &f.Destination, &f.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("forwarder not found")
	}
	if _, err := s.DB.Exec(`DELETE FROM email_forwarders WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &f, nil
}

// GetAllForwardersByDomain returns all forwarders for building the Postfix virtual alias map.
func (s *Service) GetAllForwardersByDomain(domainName string, domainID int64) ([]Forwarder, error) {
	return s.ListForwarders(domainID)
}
