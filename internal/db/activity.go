package db

import (
	"database/sql"
	"fmt"
)

// LogActivity records an action in the activity_log table.
func LogActivity(db *sql.DB, adminID int64, action, targetType string, targetID int64, details, ipAddress string) error {
	_, err := db.Exec(
		`INSERT INTO activity_log (admin_id, action, target_type, target_id, details, ip_address) VALUES (?, ?, ?, ?, ?, ?)`,
		adminID, action, nullableString(targetType), nullableInt(targetID), nullableString(details), nullableString(ipAddress),
	)
	if err != nil {
		return fmt.Errorf("logging activity: %w", err)
	}
	return nil
}

// RecentActivity returns the last N activity log entries.
func RecentActivity(db *sql.DB, limit int) ([]ActivityEntry, error) {
	rows, err := db.Query(
		`SELECT al.id, al.admin_id, a.username, al.action, al.target_type, al.target_id, al.details, al.ip_address, al.created_at
		 FROM activity_log al
		 JOIN admins a ON a.id = al.admin_id
		 ORDER BY al.created_at DESC
		 LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying activity log: %w", err)
	}
	defer rows.Close()

	var entries []ActivityEntry
	for rows.Next() {
		var e ActivityEntry
		var targetType, details, ipAddress sql.NullString
		var targetID sql.NullInt64
		if err := rows.Scan(&e.ID, &e.AdminID, &e.Username, &e.Action, &targetType, &targetID, &details, &ipAddress, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning activity log: %w", err)
		}
		e.TargetType = targetType.String
		e.TargetID = targetID.Int64
		e.Details = details.String
		e.IPAddress = ipAddress.String
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ActivityEntry represents a single activity log row.
type ActivityEntry struct {
	ID         int64  `json:"id"`
	AdminID    int64  `json:"admin_id"`
	Username   string `json:"username"`
	Action     string `json:"action"`
	TargetType string `json:"target_type,omitempty"`
	TargetID   int64  `json:"target_id,omitempty"`
	Details    string `json:"details,omitempty"`
	IPAddress  string `json:"ip_address,omitempty"`
	CreatedAt  string `json:"created_at"`
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(n int64) interface{} {
	if n == 0 {
		return nil
	}
	return n
}
