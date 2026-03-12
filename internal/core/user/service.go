// Package user provides user management operations.
package user

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{2,31}$`)

// User represents a row in the admins table.
type User struct {
	ID             int64   `json:"id"`
	Username       string  `json:"username"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	SystemUsername *string `json:"system_username,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// UserWithStats includes resource counts.
type UserWithStats struct {
	User
	DomainCount   int `json:"domain_count"`
	DatabaseCount int `json:"database_count"`
	FTPCount      int `json:"ftp_count"`
}

// Service provides user-related database operations.
type Service struct {
	DB *sql.DB
}

const userColumns = "id, username, email, role, status, system_username, created_at, updated_at"

func scanUser(row interface{ Scan(...interface{}) error }) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.SystemUsername, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// List returns all users with optional search filtering.
func (s *Service) List(search string) ([]UserWithStats, error) {
	where := "1=1"
	args := []interface{}{}
	if search != "" {
		where = "(username LIKE ? OR email LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	query := fmt.Sprintf(`
		SELECT a.%s,
			COALESCE((SELECT COUNT(*) FROM domains WHERE admin_id = a.id), 0) AS domain_count,
			COALESCE((SELECT COUNT(*) FROM databases WHERE admin_id = a.id), 0) AS database_count,
			COALESCE((SELECT COUNT(*) FROM ftp_accounts WHERE admin_id = a.id), 0) AS ftp_count
		FROM admins a
		WHERE %s
		ORDER BY a.id
	`, userColumns, where)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []UserWithStats
	for rows.Next() {
		var u UserWithStats
		err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &u.Status, &u.SystemUsername, &u.CreatedAt, &u.UpdatedAt,
			&u.DomainCount, &u.DatabaseCount, &u.FTPCount)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetByID returns a single user by ID.
func (s *Service) GetByID(id int64) (*User, error) {
	row := s.DB.QueryRow(
		fmt.Sprintf("SELECT %s FROM admins WHERE id = ?", userColumns), id,
	)
	u, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("getting user by id: %w", err)
	}
	return u, nil
}

// Create inserts a new user.
func (s *Service) Create(username, email, passwordHash, role string) (*User, error) {
	if !usernameRe.MatchString(username) {
		return nil, fmt.Errorf("invalid username: must be 3-32 characters, start with letter, contain only letters/numbers/hyphens/underscores")
	}

	validRoles := map[string]bool{"super_admin": true, "admin": true, "user": true}
	if !validRoles[role] {
		return nil, fmt.Errorf("invalid role: must be super_admin, admin, or user")
	}

	// Check uniqueness
	var exists int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM admins WHERE username = ? OR email = ?", username, email).Scan(&exists); err != nil {
		return nil, fmt.Errorf("checking duplicate user: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("username or email already exists")
	}

	// Generate system username: super_admin uses www-data, others get pp_ prefix
	sysUsername := "www-data"
	if role != "super_admin" {
		sysUsername = "pp_" + strings.ToLower(username)
	}

	result, err := s.DB.Exec(
		"INSERT INTO admins (username, email, password_hash, role, status, system_username) VALUES (?, ?, ?, ?, 'active', ?)",
		username, email, passwordHash, role, sysUsername,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update modifies a user's email and/or role.
func (s *Service) Update(id int64, email, role string) (*User, error) {
	if role != "" {
		validRoles := map[string]bool{"super_admin": true, "admin": true, "user": true}
		if !validRoles[role] {
			return nil, fmt.Errorf("invalid role: must be super_admin, admin, or user")
		}
	}

	if email != "" {
		var exists int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM admins WHERE email = ? AND id != ?", email, id).Scan(&exists); err != nil {
			return nil, fmt.Errorf("checking duplicate email: %w", err)
		}
		if exists > 0 {
			return nil, fmt.Errorf("email already in use")
		}
	}

	if email != "" && role != "" {
		_, err := s.DB.Exec("UPDATE admins SET email = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", email, role, id)
		if err != nil {
			return nil, fmt.Errorf("updating user: %w", err)
		}
	} else if email != "" {
		_, err := s.DB.Exec("UPDATE admins SET email = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", email, id)
		if err != nil {
			return nil, fmt.Errorf("updating user email: %w", err)
		}
	} else if role != "" {
		_, err := s.DB.Exec("UPDATE admins SET role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", role, id)
		if err != nil {
			return nil, fmt.Errorf("updating user role: %w", err)
		}
	}

	return s.GetByID(id)
}

// UpdatePassword sets a new password hash.
func (s *Service) UpdatePassword(id int64, passwordHash string) error {
	_, err := s.DB.Exec("UPDATE admins SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", passwordHash, id)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	return nil
}

// Suspend sets a user's status to suspended.
func (s *Service) Suspend(id int64) error {
	_, err := s.DB.Exec("UPDATE admins SET status = 'suspended', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("suspending user: %w", err)
	}
	return nil
}

// Activate sets a user's status to active.
func (s *Service) Activate(id int64) error {
	_, err := s.DB.Exec("UPDATE admins SET status = 'active', updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("activating user: %w", err)
	}
	return nil
}

// Delete removes a user from the database.
func (s *Service) Delete(id int64) error {
	// Prevent deleting the last super_admin
	var superCount int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM admins WHERE role = 'super_admin'").Scan(&superCount); err != nil {
		return fmt.Errorf("counting super admins: %w", err)
	}

	var userRole string
	if err := s.DB.QueryRow("SELECT role FROM admins WHERE id = ?", id).Scan(&userRole); err != nil {
		return fmt.Errorf("checking user role: %w", err)
	}

	if userRole == "super_admin" && superCount <= 1 {
		return fmt.Errorf("cannot delete the last super admin")
	}

	_, err := s.DB.Exec("DELETE FROM admins WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	return nil
}

// GetSystemUsername returns the system username for a given admin ID.
func (s *Service) GetSystemUsername(adminID int64) (string, error) {
	var sysUser sql.NullString
	err := s.DB.QueryRow("SELECT system_username FROM admins WHERE id = ?", adminID).Scan(&sysUser)
	if err != nil {
		return "", fmt.Errorf("getting system username: %w", err)
	}
	if !sysUser.Valid || sysUser.String == "" {
		return "www-data", nil
	}
	return sysUser.String, nil
}

// Count returns the total number of users.
func (s *Service) Count() (int, error) {
	var count int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}
