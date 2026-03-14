package app

import (
	"database/sql"
	"fmt"
	"time"
)

// InstalledApp represents an installed application in the panel.
type InstalledApp struct {
	ID           int64   `json:"id"`
	DomainID     int64   `json:"domain_id"`
	AppType      string  `json:"app_type"`
	AppName      string  `json:"app_name"`
	Version      string  `json:"version"`
	InstallPath  string  `json:"install_path"`
	DBName       *string `json:"db_name"`
	DBUser       *string `json:"db_user"`
	AdminURL     *string `json:"admin_url"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message"`
	InstallLog   *string `json:"install_log"`
	InstalledAt  string  `json:"installed_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// Service provides installed-app database operations.
type Service struct {
	DB *sql.DB
}

const appColumns = "id, domain_id, app_type, app_name, version, install_path, db_name, db_user, admin_url, status, error_message, install_log, installed_at, updated_at"

func scanApp(row interface{ Scan(...interface{}) error }) (*InstalledApp, error) {
	a := &InstalledApp{}
	err := row.Scan(&a.ID, &a.DomainID, &a.AppType, &a.AppName, &a.Version, &a.InstallPath,
		&a.DBName, &a.DBUser, &a.AdminURL, &a.Status, &a.ErrorMessage, &a.InstallLog,
		&a.InstalledAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) ListByDomain(domainID int64) ([]InstalledApp, error) {
	rows, err := s.DB.Query(
		`SELECT `+appColumns+` FROM installed_apps WHERE domain_id = ? ORDER BY installed_at DESC`, domainID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []InstalledApp
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *a)
	}
	return apps, rows.Err()
}

func (s *Service) GetByID(id int64) (*InstalledApp, error) {
	row := s.DB.QueryRow(`SELECT `+appColumns+` FROM installed_apps WHERE id = ?`, id)
	a, err := scanApp(row)
	if err != nil {
		return nil, fmt.Errorf("app not found")
	}
	return a, nil
}

func (s *Service) Create(domainID int64, appType, appName, version, installPath string, dbName, dbUser *string) (*InstalledApp, error) {
	res, err := s.DB.Exec(
		`INSERT INTO installed_apps (domain_id, app_type, app_name, version, install_path, db_name, db_user, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')`,
		domainID, appType, appName, version, installPath, dbName, dbUser,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetByID(id)
}

func (s *Service) UpdateStatus(id int64, status string, errMsg *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(
		`UPDATE installed_apps SET status = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		status, errMsg, now, id,
	)
	return err
}

func (s *Service) UpdateVersion(id int64, version string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(
		`UPDATE installed_apps SET version = ?, updated_at = ? WHERE id = ?`,
		version, now, id,
	)
	return err
}

func (s *Service) SetAdminURL(id int64, adminURL string) error {
	_, err := s.DB.Exec(
		`UPDATE installed_apps SET admin_url = ? WHERE id = ?`,
		adminURL, id,
	)
	return err
}

func (s *Service) AppendLog(id int64, line string) error {
	_, err := s.DB.Exec(
		`UPDATE installed_apps SET install_log = COALESCE(install_log, '') || ? || char(10) WHERE id = ?`,
		line, id,
	)
	return err
}

func (s *Service) Delete(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM installed_apps WHERE id = ?`, id)
	return err
}
