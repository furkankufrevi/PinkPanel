// Package dns provides the DNS record management service layer.
package dns

import (
	"database/sql"
	"fmt"
	"net"
	"regexp"
)

// allowedTypes lists the DNS record types accepted by the system.
var allowedTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CNAME": true,
	"MX":    true,
	"TXT":   true,
	"NS":    true,
	"SOA":   true,
	"SRV":   true,
	"CAA":   true,
}

// hostnameRe validates hostnames used in CNAME, MX, and NS records.
var hostnameRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}\.?$`)

// Record represents a row in the dns_records table.
type Record struct {
	ID        int64  `json:"id"`
	DomainID  int64  `json:"domain_id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	TTL       int    `json:"ttl"`
	Priority  *int   `json:"priority"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Service provides DNS-record-related database operations.
type Service struct {
	DB *sql.DB
}

// ListByDomain returns all DNS records for a domain, ordered by type then name.
func (s *Service) ListByDomain(domainID int64) ([]Record, error) {
	rows, err := s.DB.Query(
		"SELECT id, domain_id, type, name, value, ttl, priority, created_at, updated_at FROM dns_records WHERE domain_id = ? ORDER BY type, name",
		domainID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing dns records: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.DomainID, &r.Type, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning dns record row: %w", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating dns record rows: %w", err)
	}

	return records, nil
}

// GetByID returns a single DNS record by its primary key.
func (s *Service) GetByID(id int64) (*Record, error) {
	r := &Record{}
	err := s.DB.QueryRow(
		"SELECT id, domain_id, type, name, value, ttl, priority, created_at, updated_at FROM dns_records WHERE id = ?", id,
	).Scan(&r.ID, &r.DomainID, &r.Type, &r.Name, &r.Value, &r.TTL, &r.Priority, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting dns record by id: %w", err)
	}
	return r, nil
}

// validateRecord checks that the record type is allowed and that the value is
// valid for the given type.
func validateRecord(recType, value string) error {
	if !allowedTypes[recType] {
		return fmt.Errorf("invalid record type: %s", recType)
	}

	switch recType {
	case "A":
		ip := net.ParseIP(value)
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid IPv4 address for A record: %s", value)
		}
	case "AAAA":
		ip := net.ParseIP(value)
		if ip == nil || ip.To4() != nil {
			return fmt.Errorf("invalid IPv6 address for AAAA record: %s", value)
		}
	case "CNAME", "MX", "NS":
		if !hostnameRe.MatchString(value) {
			return fmt.Errorf("invalid hostname for %s record: %s", recType, value)
		}
	case "TXT":
		if value == "" {
			return fmt.Errorf("TXT record value must not be empty")
		}
	// SOA, SRV, CAA — accept any non-empty value.
	default:
		if value == "" {
			return fmt.Errorf("%s record value must not be empty", recType)
		}
	}

	return nil
}

// Create validates the inputs, inserts a new DNS record, and returns it.
func (s *Service) Create(domainID int64, recType, name, value string, ttl int, priority *int) (*Record, error) {
	if err := validateRecord(recType, value); err != nil {
		return nil, err
	}

	result, err := s.DB.Exec(
		"INSERT INTO dns_records (domain_id, type, name, value, ttl, priority) VALUES (?, ?, ?, ?, ?, ?)",
		domainID, recType, name, value, ttl, priority,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting dns record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update validates the inputs, updates an existing DNS record, and returns it.
func (s *Service) Update(id int64, recType, name, value string, ttl int, priority *int) (*Record, error) {
	if err := validateRecord(recType, value); err != nil {
		return nil, err
	}

	_, err := s.DB.Exec(
		"UPDATE dns_records SET type = ?, name = ?, value = ?, ttl = ?, priority = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		recType, name, value, ttl, priority, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating dns record: %w", err)
	}

	return s.GetByID(id)
}

// Delete removes a DNS record from the database.
func (s *Service) Delete(id int64) error {
	_, err := s.DB.Exec("DELETE FROM dns_records WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting dns record: %w", err)
	}
	return nil
}

// DeleteByDomain removes all DNS records for a given domain.
func (s *Service) DeleteByDomain(domainID int64) error {
	_, err := s.DB.Exec("DELETE FROM dns_records WHERE domain_id = ?", domainID)
	if err != nil {
		return fmt.Errorf("deleting dns records for domain: %w", err)
	}
	return nil
}

// DeleteByName removes DNS records matching a domain and record name.
func (s *Service) DeleteByName(domainID int64, name string) error {
	_, err := s.DB.Exec("DELETE FROM dns_records WHERE domain_id = ? AND name = ?", domainID, name)
	if err != nil {
		return fmt.Errorf("deleting dns record by name: %w", err)
	}
	return nil
}

// CreateDefaultRecords inserts the standard set of DNS records for a newly
// created domain.
func (s *Service) CreateDefaultRecords(domainID int64, domainName, serverIP string) error {
	mx10 := 10

	defaults := []struct {
		recType  string
		name     string
		value    string
		ttl      int
		priority *int
	}{
		// SOA — mname + rname only; serial/timers are rendered by the zone template
		{"SOA", "@", fmt.Sprintf("ns1.%s. admin.%s.", domainName, domainName), 86400, nil},
		// Nameservers
		{"NS", "@", fmt.Sprintf("ns1.%s.", domainName), 86400, nil},
		{"NS", "@", fmt.Sprintf("ns2.%s.", domainName), 86400, nil},
		// Glue records — so ns1/ns2 actually resolve
		{"A", "ns1", serverIP, 3600, nil},
		{"A", "ns2", serverIP, 3600, nil},
		// Domain itself
		{"A", "@", serverIP, 3600, nil},
		{"A", "www", serverIP, 3600, nil},
		// Mail
		{"A", "mail", serverIP, 3600, nil},
		{"MX", "@", fmt.Sprintf("mail.%s.", domainName), 3600, &mx10},
	}

	for _, d := range defaults {
		_, err := s.DB.Exec(
			"INSERT INTO dns_records (domain_id, type, name, value, ttl, priority) VALUES (?, ?, ?, ?, ?, ?)",
			domainID, d.recType, d.name, d.value, d.ttl, d.priority,
		)
		if err != nil {
			return fmt.Errorf("inserting default %s record: %w", d.recType, err)
		}
	}

	return nil
}
