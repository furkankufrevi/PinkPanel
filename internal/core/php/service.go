package php

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
)

var allowedVersionRe = regexp.MustCompile(`^(7\.4|8\.[0-4])$`)

type PHPSettings struct {
	Version  string            `json:"version"`
	Settings map[string]string `json:"settings"`
}

type PHPPoolConfig struct {
	Domain       string
	User         string
	Group        string
	ListenSocket string
	PHPVersion   string
	Settings     map[string]string
}

type Service struct {
	DB *sql.DB
}

// GetDomainPHP returns PHP settings for a domain.
func (s *Service) GetDomainPHP(domainID int64) (*PHPSettings, error) {
	var version string
	var settingsJSON sql.NullString
	err := s.DB.QueryRow("SELECT php_version, php_settings FROM domains WHERE id = ?", domainID).Scan(&version, &settingsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	if err != nil {
		return nil, err
	}
	settings := make(map[string]string)
	if settingsJSON.Valid && settingsJSON.String != "" && settingsJSON.String != "{}" {
		if err := json.Unmarshal([]byte(settingsJSON.String), &settings); err != nil {
			return nil, fmt.Errorf("invalid php settings JSON: %w", err)
		}
	}
	return &PHPSettings{Version: version, Settings: settings}, nil
}

// UpdateDomainPHP updates PHP version and settings for a domain.
func (s *Service) UpdateDomainPHP(domainID int64, version string, settings map[string]string) (*PHPSettings, error) {
	if !allowedVersionRe.MatchString(version) {
		return nil, fmt.Errorf("invalid PHP version: %s", version)
	}
	if settings == nil {
		settings = make(map[string]string)
	}
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	res, err := s.DB.Exec(
		"UPDATE domains SET php_version = ?, php_settings = ?, updated_at = datetime('now') WHERE id = ?",
		version, string(settingsJSON), domainID,
	)
	if err != nil {
		return nil, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("domain not found")
	}
	return &PHPSettings{Version: version, Settings: settings}, nil
}

// ListInstalledVersions returns available PHP versions.
func (s *Service) ListInstalledVersions() []string {
	return []string{"8.4", "8.3", "8.2", "8.1", "8.0", "7.4"}
}

// DefaultPoolConfig returns a PHPPoolConfig with sensible defaults.
func DefaultPoolConfig(domainName, phpVersion string, settings map[string]string) *PHPPoolConfig {
	return &PHPPoolConfig{
		Domain:       domainName,
		User:         "www-data",
		Group:        "www-data",
		ListenSocket: fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, domainName),
		PHPVersion:   phpVersion,
		Settings:     settings,
	}
}
