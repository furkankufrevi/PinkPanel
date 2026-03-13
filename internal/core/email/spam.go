package email

import "fmt"

// SpamSettings holds per-domain spam filtering configuration.
type SpamSettings struct {
	ID             int64   `json:"id"`
	DomainID       int64   `json:"domain_id"`
	Enabled        bool    `json:"enabled"`
	ScoreThreshold float64 `json:"score_threshold"`
	Action         string  `json:"action"` // "mark", "junk", "delete"
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// SpamListEntry represents a whitelist or blacklist entry.
type SpamListEntry struct {
	ID        int64  `json:"id"`
	DomainID  int64  `json:"domain_id"`
	ListType  string `json:"list_type"` // "whitelist" or "blacklist"
	Entry     string `json:"entry"`
	CreatedAt string `json:"created_at"`
}

// GetSpamSettings returns spam settings for a domain (creates defaults if none exist).
func (s *Service) GetSpamSettings(domainID int64) (*SpamSettings, error) {
	var ss SpamSettings
	var enabled int
	err := s.DB.QueryRow(
		`SELECT id, domain_id, enabled, score_threshold, action, created_at, updated_at
		 FROM email_spam_settings WHERE domain_id = ?`, domainID,
	).Scan(&ss.ID, &ss.DomainID, &enabled, &ss.ScoreThreshold, &ss.Action, &ss.CreatedAt, &ss.UpdatedAt)
	if err != nil {
		// Return defaults
		return &SpamSettings{
			DomainID:       domainID,
			Enabled:        false,
			ScoreThreshold: 5.0,
			Action:         "mark",
		}, nil
	}
	ss.Enabled = enabled == 1
	return &ss, nil
}

// UpdateSpamSettings upserts spam settings for a domain.
func (s *Service) UpdateSpamSettings(domainID int64, enabled bool, scoreThreshold float64, action string) error {
	if action != "mark" && action != "junk" && action != "delete" {
		return fmt.Errorf("invalid action: must be mark, junk, or delete")
	}
	if scoreThreshold < 1.0 || scoreThreshold > 20.0 {
		return fmt.Errorf("score threshold must be between 1.0 and 20.0")
	}
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.DB.Exec(
		`INSERT INTO email_spam_settings (domain_id, enabled, score_threshold, action)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(domain_id) DO UPDATE SET enabled = ?, score_threshold = ?, action = ?, updated_at = CURRENT_TIMESTAMP`,
		domainID, enabledInt, scoreThreshold, action,
		enabledInt, scoreThreshold, action,
	)
	return err
}

// ListSpamEntries returns whitelist or blacklist entries for a domain.
func (s *Service) ListSpamEntries(domainID int64, listType string) ([]SpamListEntry, error) {
	rows, err := s.DB.Query(
		`SELECT id, domain_id, list_type, entry, created_at
		 FROM email_spam_lists WHERE domain_id = ? AND list_type = ? ORDER BY entry`,
		domainID, listType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SpamListEntry
	for rows.Next() {
		var e SpamListEntry
		if err := rows.Scan(&e.ID, &e.DomainID, &e.ListType, &e.Entry, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// AddSpamEntry adds a whitelist or blacklist entry.
func (s *Service) AddSpamEntry(domainID int64, listType, entry string) (*SpamListEntry, error) {
	if listType != "whitelist" && listType != "blacklist" {
		return nil, fmt.Errorf("list_type must be whitelist or blacklist")
	}
	if entry == "" {
		return nil, fmt.Errorf("entry must not be empty")
	}
	res, err := s.DB.Exec(
		`INSERT INTO email_spam_lists (domain_id, list_type, entry) VALUES (?, ?, ?)`,
		domainID, listType, entry,
	)
	if err != nil {
		return nil, fmt.Errorf("entry already exists")
	}
	id, _ := res.LastInsertId()
	var e SpamListEntry
	s.DB.QueryRow(
		`SELECT id, domain_id, list_type, entry, created_at FROM email_spam_lists WHERE id = ?`, id,
	).Scan(&e.ID, &e.DomainID, &e.ListType, &e.Entry, &e.CreatedAt)
	return &e, nil
}

// DeleteSpamEntry removes a whitelist or blacklist entry.
func (s *Service) DeleteSpamEntry(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM email_spam_lists WHERE id = ?`, id)
	return err
}
