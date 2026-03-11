package backup

import (
	"database/sql"
	"fmt"
	"time"
)

type Backup struct {
	ID          int64   `json:"id"`
	DomainID    *int64  `json:"domain_id"`
	Type        string  `json:"type"`
	FilePath    string  `json:"file_path"`
	SizeBytes   int64   `json:"size_bytes"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	CompletedAt *string `json:"completed_at"`
}

type Service struct {
	DB *sql.DB
}

func (s *Service) List(domainID *int64) ([]Backup, error) {
	query := `SELECT id, domain_id, type, file_path, size_bytes, status, created_at, completed_at FROM backups`
	var args []any
	if domainID != nil {
		query += ` WHERE domain_id = ?`
		args = append(args, *domainID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []Backup
	for rows.Next() {
		var b Backup
		if err := rows.Scan(&b.ID, &b.DomainID, &b.Type, &b.FilePath, &b.SizeBytes, &b.Status, &b.CreatedAt, &b.CompletedAt); err != nil {
			return nil, err
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

func (s *Service) GetByID(id int64) (*Backup, error) {
	var b Backup
	err := s.DB.QueryRow(
		`SELECT id, domain_id, type, file_path, size_bytes, status, created_at, completed_at FROM backups WHERE id = ?`, id,
	).Scan(&b.ID, &b.DomainID, &b.Type, &b.FilePath, &b.SizeBytes, &b.Status, &b.CreatedAt, &b.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("backup not found")
	}
	return &b, nil
}

func (s *Service) Create(domainID *int64, backupType, filePath string) (*Backup, error) {
	if backupType != "full" && backupType != "domain" {
		return nil, fmt.Errorf("invalid backup type: must be 'full' or 'domain'")
	}
	if backupType == "domain" && domainID == nil {
		return nil, fmt.Errorf("domain_id is required for domain backups")
	}

	res, err := s.DB.Exec(
		`INSERT INTO backups (domain_id, type, file_path, status) VALUES (?, ?, ?, 'pending')`,
		domainID, backupType, filePath,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) UpdateStatus(id int64, status string, sizeBytes int64) error {
	var completedAt *string
	if status == "completed" || status == "failed" {
		t := time.Now().UTC().Format(time.RFC3339)
		completedAt = &t
	}
	_, err := s.DB.Exec(
		`UPDATE backups SET status = ?, size_bytes = ?, completed_at = ? WHERE id = ?`,
		status, sizeBytes, completedAt, id,
	)
	return err
}

func (s *Service) Delete(id int64) (*Backup, error) {
	b, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec(`DELETE FROM backups WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Service) Count() (int, error) {
	var count int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM backups`).Scan(&count)
	return count, err
}
