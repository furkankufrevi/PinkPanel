package ftp

import (
	"database/sql"
	"fmt"
	"regexp"
)

var safeUsernameRe = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type Account struct {
	ID        int64  `json:"id"`
	DomainID  int64  `json:"domain_id"`
	Username  string `json:"username"`
	HomeDir   string `json:"home_dir"`
	QuotaMB   int64  `json:"quota_mb"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) List(domainID *int64) ([]Account, error) {
	query := `SELECT id, domain_id, username, home_dir, quota_mb, created_at, updated_at FROM ftp_accounts`
	var args []any
	if domainID != nil {
		query += ` WHERE domain_id = ?`
		args = append(args, *domainID)
	}
	query += ` ORDER BY username`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Username, &a.HomeDir, &a.QuotaMB, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (s *Service) GetByID(id int64) (*Account, error) {
	var a Account
	err := s.DB.QueryRow(
		`SELECT id, domain_id, username, home_dir, quota_mb, created_at, updated_at FROM ftp_accounts WHERE id = ?`, id,
	).Scan(&a.ID, &a.DomainID, &a.Username, &a.HomeDir, &a.QuotaMB, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("FTP account not found")
	}
	return &a, nil
}

func (s *Service) Create(domainID int64, username, homeDir string, quotaMB int64) (*Account, error) {
	if !safeUsernameRe.MatchString(username) {
		return nil, fmt.Errorf("invalid username: only alphanumeric, underscore, dot, and hyphen allowed")
	}
	if len(username) < 2 || len(username) > 32 {
		return nil, fmt.Errorf("username must be 2-32 characters")
	}

	res, err := s.DB.Exec(
		`INSERT INTO ftp_accounts (domain_id, username, home_dir, quota_mb) VALUES (?, ?, ?, ?)`,
		domainID, username, homeDir, quotaMB,
	)
	if err != nil {
		return nil, fmt.Errorf("username already exists or invalid domain")
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) Delete(id int64) (*Account, error) {
	a, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM ftp_accounts WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) UpdateQuota(id int64, quotaMB int64) error {
	_, err := s.DB.Exec(`UPDATE ftp_accounts SET quota_mb = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, quotaMB, id)
	return err
}

func (s *Service) Count() (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM ftp_accounts`).Scan(&count)
	return count, err
}
