package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

// DNSTemplateHandler handles DNS template operations.
type DNSTemplateHandler struct {
	DB          *sql.DB
	TemplateSvc *dns.TemplateService
	DNSSvc      *dns.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// ListTemplates returns all templates (presets + custom).
func (h *DNSTemplateHandler) ListTemplates(c *fiber.Ctx) error {
	templates, err := h.TemplateSvc.ListTemplates()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	return c.JSON(fiber.Map{"data": templates})
}

// GetTemplate returns a single template.
func (h *DNSTemplateHandler) GetTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid template ID"}})
	}

	t, err := h.TemplateSvc.GetTemplate(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(fiber.Map{"data": t})
}

// CreateTemplate creates a custom template.
func (h *DNSTemplateHandler) CreateTemplate(c *fiber.Ctx) error {
	var req struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Category    string               `json:"category"`
		Records     []dns.TemplateRecord  `json:"records"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	t, err := h.TemplateSvc.CreateTemplate(req.Name, req.Description, req.Category, req.Records)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_dns_template", "dns_template", t.ID, t.Name, c.IP())

	return c.Status(201).JSON(fiber.Map{"data": t})
}

// UpdateTemplate updates a custom template.
func (h *DNSTemplateHandler) UpdateTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid template ID"}})
	}

	var req struct {
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Category    string               `json:"category"`
		Records     []dns.TemplateRecord  `json:"records"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	t, err := h.TemplateSvc.UpdateTemplate(id, req.Name, req.Description, req.Category, req.Records)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_dns_template", "dns_template", t.ID, t.Name, c.IP())

	return c.JSON(fiber.Map{"data": t})
}

// DeleteTemplate removes a custom template.
func (h *DNSTemplateHandler) DeleteTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid template ID"}})
	}

	if err := h.TemplateSvc.DeleteTemplate(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_dns_template", "dns_template", id, "", c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// ApplyTemplate applies a template's records to a domain.
func (h *DNSTemplateHandler) ApplyTemplate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		TemplateID int64  `json:"template_id"`
		Mode       string `json:"mode"` // "merge" or "replace"
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Mode == "" {
		req.Mode = "merge"
	}

	d, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, d) {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": "access denied"}})
	}

	t, err := h.TemplateSvc.GetTemplate(req.TemplateID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "template not found"}})
	}

	// Resolve variables
	serverIP := getServerIP()
	serverIPv6 := getServerIPv6()
	hostname, _ := getHostname()
	resolved := dns.ResolveVariables(t.Records, d.Name, serverIP, serverIPv6, hostname)

	if req.Mode == "replace" {
		// Delete all existing records
		if err := h.DNSSvc.DeleteByDomain(domainID); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": "failed to clear existing records"}})
		}
	}

	// Insert resolved records
	for _, r := range resolved {
		_, err := h.DNSSvc.Create(domainID, r.Type, r.Name, r.Value, r.TTL, r.Priority)
		if err != nil {
			// Skip validation errors in merge mode (e.g., duplicate records)
			if req.Mode == "merge" {
				continue
			}
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": fmt.Sprintf("failed to create record: %v", err)}})
		}
	}

	// Regenerate zone
	dnsHandler := &DNSHandler{
		DB:          h.DB,
		DNSSvc:      h.DNSSvc,
		DomainSvc:   h.DomainSvc,
		AgentClient: h.AgentClient,
	}
	zoneWarn := regenerateZone(dnsHandler, domainID, d.Name)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "apply_dns_template", "domain", domainID, fmt.Sprintf("%s (%s)", t.Name, req.Mode), c.IP())

	// Return updated records
	records, _ := h.DNSSvc.ListByDomain(domainID)
	if records == nil {
		records = []dns.Record{}
	}

	resp := fiber.Map{"data": records, "message": fmt.Sprintf("Template '%s' applied (%s mode)", t.Name, req.Mode)}
	if zoneWarn != "" {
		resp["warning"] = zoneWarn
	}
	return c.JSON(resp)
}

// SaveAsTemplate saves a domain's current DNS records as a new template.
func (h *DNSTemplateHandler) SaveAsTemplate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	d, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, d) {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": "access denied"}})
	}

	t, err := h.TemplateSvc.SaveDomainAsTemplate(h.DNSSvc, domainID, d.Name, req.Name, req.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "save_dns_template", "dns_template", t.ID, t.Name, c.IP())

	return c.Status(201).JSON(fiber.Map{"data": t})
}

// ExportTemplate returns template JSON for download.
func (h *DNSTemplateHandler) ExportTemplate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid template ID"}})
	}

	data, err := h.TemplateSvc.ExportTemplate(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=dns-template.json")
	return c.Send(data)
}

// ImportTemplate creates a template from uploaded JSON.
func (h *DNSTemplateHandler) ImportTemplate(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "empty request body"}})
	}

	t, err := h.TemplateSvc.ImportTemplate(body)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "import_dns_template", "dns_template", t.ID, t.Name, c.IP())

	return c.Status(201).JSON(fiber.Map{"data": t})
}

func getHostname() (string, error) {
	return os.Hostname()
}
