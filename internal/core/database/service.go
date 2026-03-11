package database

import (
	"database/sql"
	"fmt"
	"regexp"
)

var safeNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

type Database struct {
	ID        int64  `json:"id"`
	DomainID  *int64 `json:"domain_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type DatabaseUser struct {
	ID          int64  `json:"id"`
	DatabaseID  int64  `json:"database_id"`
	Username    string `json:"username"`
	Host        string `json:"host"`
	Permissions string `json:"permissions"`
	CreatedAt   string `json:"created_at"`
}

type Service struct {
	DB *sql.DB
}

// List returns all databases, optionally filtered by domain.
func (s *Service) List(domainID *int64) ([]Database, error) {
	query := "SELECT id, domain_id, name, type, size_bytes, created_at, updated_at FROM databases"
	var args []interface{}
	if domainID != nil {
		query += " WHERE domain_id = ?"
		args = append(args, *domainID)
	}
	query += " ORDER BY name"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing databases: %w", err)
	}
	defer rows.Close()

	var dbs []Database
	for rows.Next() {
		var d Database
		if err := rows.Scan(&d.ID, &d.DomainID, &d.Name, &d.Type, &d.SizeBytes, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

// GetByID returns a database by ID.
func (s *Service) GetByID(id int64) (*Database, error) {
	d := &Database{}
	err := s.DB.QueryRow(
		"SELECT id, domain_id, name, type, size_bytes, created_at, updated_at FROM databases WHERE id = ?", id,
	).Scan(&d.ID, &d.DomainID, &d.Name, &d.Type, &d.SizeBytes, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("database not found")
	}
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Create inserts a new database record.
func (s *Service) Create(name string, domainID *int64) (*Database, error) {
	if !safeNameRe.MatchString(name) {
		return nil, fmt.Errorf("invalid database name: must be alphanumeric with underscores/hyphens, max 64 chars")
	}
	res, err := s.DB.Exec(
		"INSERT INTO databases (name, domain_id) VALUES (?, ?)", name, domainID,
	)
	if err != nil {
		return nil, fmt.Errorf("creating database: %w", err)
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

// Delete removes a database record.
func (s *Service) Delete(id int64) error {
	res, err := s.DB.Exec("DELETE FROM databases WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting database: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("database not found")
	}
	return nil
}

// UpdateSize updates the size_bytes for a database.
func (s *Service) UpdateSize(id int64, sizeBytes int64) error {
	_, err := s.DB.Exec("UPDATE databases SET size_bytes = ?, updated_at = datetime('now') WHERE id = ?", sizeBytes, id)
	return err
}

// ListUsers returns all users for a database.
func (s *Service) ListUsers(databaseID int64) ([]DatabaseUser, error) {
	rows, err := s.DB.Query(
		"SELECT id, database_id, username, host, permissions, created_at FROM database_users WHERE database_id = ? ORDER BY username",
		databaseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []DatabaseUser
	for rows.Next() {
		var u DatabaseUser
		if err := rows.Scan(&u.ID, &u.DatabaseID, &u.Username, &u.Host, &u.Permissions, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// CreateUser inserts a new database user record.
func (s *Service) CreateUser(databaseID int64, username, host, permissions string) (*DatabaseUser, error) {
	if !safeNameRe.MatchString(username) {
		return nil, fmt.Errorf("invalid username: must be alphanumeric with underscores/hyphens")
	}
	if host == "" {
		host = "localhost"
	}
	if permissions == "" {
		permissions = "ALL"
	}
	res, err := s.DB.Exec(
		"INSERT INTO database_users (database_id, username, host, permissions) VALUES (?, ?, ?, ?)",
		databaseID, username, host, permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	id, _ := res.LastInsertId()
	u := &DatabaseUser{}
	err = s.DB.QueryRow(
		"SELECT id, database_id, username, host, permissions, created_at FROM database_users WHERE id = ?", id,
	).Scan(&u.ID, &u.DatabaseID, &u.Username, &u.Host, &u.Permissions, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// DeleteUser removes a database user record.
func (s *Service) DeleteUser(id int64) (*DatabaseUser, error) {
	u := &DatabaseUser{}
	err := s.DB.QueryRow(
		"SELECT id, database_id, username, host, permissions, created_at FROM database_users WHERE id = ?", id,
	).Scan(&u.ID, &u.DatabaseID, &u.Username, &u.Host, &u.Permissions, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.DB.Exec("DELETE FROM database_users WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("deleting user: %w", err)
	}
	return u, nil
}

// Count returns total number of databases.
func (s *Service) Count() (int, error) {
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM databases").Scan(&count)
	return count, err
}
