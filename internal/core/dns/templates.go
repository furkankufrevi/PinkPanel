package dns

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Template represents a DNS template (custom or preset).
type Template struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	IsPreset    bool             `json:"is_preset"`
	Records     []TemplateRecord `json:"records"`
	CreatedAt   string           `json:"created_at,omitempty"`
	UpdatedAt   string           `json:"updated_at,omitempty"`
}

// TemplateRecord represents a DNS record within a template.
type TemplateRecord struct {
	ID         int64  `json:"id"`
	TemplateID int64  `json:"template_id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	TTL        int    `json:"ttl"`
	Priority   *int   `json:"priority"`
}

// TemplateExport is the JSON format for import/export.
type TemplateExport struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Category    string                `json:"category"`
	Records     []TemplateRecordExport `json:"records"`
}

// TemplateRecordExport is a record in export format (no IDs).
type TemplateRecordExport struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority,omitempty"`
}

// TemplateService provides DNS template operations.
type TemplateService struct {
	DB *sql.DB
}

// ListTemplates returns all custom templates plus presets.
func (s *TemplateService) ListTemplates() ([]Template, error) {
	// Start with presets
	templates := []Template{}
	for _, p := range Presets() {
		t := Template{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Category:    p.Category,
			IsPreset:    true,
			Records:     make([]TemplateRecord, len(p.Records)),
		}
		for i, r := range p.Records {
			t.Records[i] = TemplateRecord{
				Type:     r.Type,
				Name:     r.Name,
				Value:    r.Value,
				TTL:      r.TTL,
				Priority: r.Priority,
			}
		}
		templates = append(templates, t)
	}

	// Add custom templates from DB
	rows, err := s.DB.Query("SELECT id, name, description, category, created_at, updated_at FROM dns_templates ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning template: %w", err)
		}
		t.Records, _ = s.getTemplateRecords(t.ID)
		templates = append(templates, t)
	}

	return templates, rows.Err()
}

// GetTemplate returns a single template by ID. Negative IDs are presets.
func (s *TemplateService) GetTemplate(id int64) (*Template, error) {
	if id < 0 {
		for _, p := range Presets() {
			if p.ID == id {
				t := &Template{
					ID:          p.ID,
					Name:        p.Name,
					Description: p.Description,
					Category:    p.Category,
					IsPreset:    true,
					Records:     make([]TemplateRecord, len(p.Records)),
				}
				for i, r := range p.Records {
					t.Records[i] = TemplateRecord{
						Type: r.Type, Name: r.Name, Value: r.Value, TTL: r.TTL, Priority: r.Priority,
					}
				}
				return t, nil
			}
		}
		return nil, fmt.Errorf("preset template not found")
	}

	t := &Template{}
	err := s.DB.QueryRow(
		"SELECT id, name, description, category, created_at, updated_at FROM dns_templates WHERE id = ?", id,
	).Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting template: %w", err)
	}
	t.Records, _ = s.getTemplateRecords(t.ID)
	return t, nil
}

// CreateTemplate creates a custom template with records.
func (s *TemplateService) CreateTemplate(name, description, category string, records []TemplateRecord) (*Template, error) {
	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if category == "" {
		category = "custom"
	}

	result, err := s.DB.Exec(
		"INSERT INTO dns_templates (name, description, category) VALUES (?, ?, ?)",
		name, description, category,
	)
	if err != nil {
		return nil, fmt.Errorf("creating template: %w", err)
	}

	id, _ := result.LastInsertId()

	for _, r := range records {
		_, err := s.DB.Exec(
			"INSERT INTO dns_template_records (template_id, type, name, value, ttl, priority) VALUES (?, ?, ?, ?, ?, ?)",
			id, r.Type, r.Name, r.Value, r.TTL, r.Priority,
		)
		if err != nil {
			return nil, fmt.Errorf("inserting template record: %w", err)
		}
	}

	return s.GetTemplate(id)
}

// UpdateTemplate updates a custom template.
func (s *TemplateService) UpdateTemplate(id int64, name, description, category string, records []TemplateRecord) (*Template, error) {
	if id < 0 {
		return nil, fmt.Errorf("cannot modify preset templates")
	}

	_, err := s.DB.Exec(
		"UPDATE dns_templates SET name = ?, description = ?, category = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		name, description, category, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating template: %w", err)
	}

	// Replace all records
	if _, err := s.DB.Exec("DELETE FROM dns_template_records WHERE template_id = ?", id); err != nil {
		return nil, fmt.Errorf("clearing template records: %w", err)
	}
	for _, r := range records {
		_, err := s.DB.Exec(
			"INSERT INTO dns_template_records (template_id, type, name, value, ttl, priority) VALUES (?, ?, ?, ?, ?, ?)",
			id, r.Type, r.Name, r.Value, r.TTL, r.Priority,
		)
		if err != nil {
			return nil, fmt.Errorf("inserting template record: %w", err)
		}
	}

	return s.GetTemplate(id)
}

// DeleteTemplate removes a custom template.
func (s *TemplateService) DeleteTemplate(id int64) error {
	if id < 0 {
		return fmt.Errorf("cannot delete preset templates")
	}
	_, err := s.DB.Exec("DELETE FROM dns_templates WHERE id = ?", id)
	return err
}

// SaveDomainAsTemplate saves a domain's current DNS records as a custom template.
func (s *TemplateService) SaveDomainAsTemplate(dnsSvc *Service, domainID int64, domainName, templateName, description string) (*Template, error) {
	records, err := dnsSvc.ListByDomain(domainID)
	if err != nil {
		return nil, fmt.Errorf("listing domain records: %w", err)
	}

	var tmplRecords []TemplateRecord
	for _, r := range records {
		// Convert domain-specific values back to variables
		value := r.Value
		name := r.Name

		// Don't templatize — keep raw values for saved templates
		tmplRecords = append(tmplRecords, TemplateRecord{
			Type:     r.Type,
			Name:     name,
			Value:    value,
			TTL:      r.TTL,
			Priority: r.Priority,
		})
	}

	return s.CreateTemplate(templateName, description, "custom", tmplRecords)
}

// ResolveVariables replaces template variables in records with actual values.
func ResolveVariables(records []TemplateRecord, domain, ip, ipv6, hostname string) []TemplateRecord {
	resolved := make([]TemplateRecord, len(records))
	for i, r := range records {
		resolved[i] = TemplateRecord{
			Type:     r.Type,
			Name:     replaceVars(r.Name, domain, ip, ipv6, hostname),
			Value:    replaceVars(r.Value, domain, ip, ipv6, hostname),
			TTL:      r.TTL,
			Priority: r.Priority,
		}
	}
	return resolved
}

func replaceVars(s, domain, ip, ipv6, hostname string) string {
	s = strings.ReplaceAll(s, "{{domain}}", domain)
	s = strings.ReplaceAll(s, "{{ip}}", ip)
	s = strings.ReplaceAll(s, "{{ipv6}}", ipv6)
	s = strings.ReplaceAll(s, "{{hostname}}", hostname)
	return s
}

// ExportTemplate returns a template in export JSON format.
func (s *TemplateService) ExportTemplate(id int64) ([]byte, error) {
	t, err := s.GetTemplate(id)
	if err != nil {
		return nil, err
	}

	export := TemplateExport{
		Name:        t.Name,
		Description: t.Description,
		Category:    t.Category,
		Records:     make([]TemplateRecordExport, len(t.Records)),
	}
	for i, r := range t.Records {
		export.Records[i] = TemplateRecordExport{
			Type: r.Type, Name: r.Name, Value: r.Value, TTL: r.TTL, Priority: r.Priority,
		}
	}

	return json.MarshalIndent(export, "", "  ")
}

// ImportTemplate creates a template from exported JSON.
func (s *TemplateService) ImportTemplate(data []byte) (*Template, error) {
	var export TemplateExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("invalid template JSON: %w", err)
	}

	if export.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}

	records := make([]TemplateRecord, len(export.Records))
	for i, r := range export.Records {
		records[i] = TemplateRecord{
			Type: r.Type, Name: r.Name, Value: r.Value, TTL: r.TTL, Priority: r.Priority,
		}
	}

	return s.CreateTemplate(export.Name, export.Description, export.Category, records)
}

func (s *TemplateService) getTemplateRecords(templateID int64) ([]TemplateRecord, error) {
	rows, err := s.DB.Query(
		"SELECT id, template_id, type, name, value, ttl, priority FROM dns_template_records WHERE template_id = ? ORDER BY type, name",
		templateID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TemplateRecord
	for rows.Next() {
		var r TemplateRecord
		if err := rows.Scan(&r.ID, &r.TemplateID, &r.Type, &r.Name, &r.Value, &r.TTL, &r.Priority); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	if records == nil {
		records = []TemplateRecord{}
	}
	return records, rows.Err()
}
