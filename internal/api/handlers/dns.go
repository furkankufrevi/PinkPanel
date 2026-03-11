package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// DNSHandler handles DNS record CRUD operations.
type DNSHandler struct {
	DB          *sql.DB
	DNSSvc      *dns.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

type createDNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority"`
}

type updateDNSRecordRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority *int   `json:"priority"`
}

// ListRecords returns all DNS records for a domain.
func (h *DNSHandler) ListRecords(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	// Validate domain exists
	_, err = h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	records, err := h.DNSSvc.ListByDomain(domainID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list DNS records",
			},
		})
	}

	if records == nil {
		records = []dns.Record{}
	}

	return c.JSON(fiber.Map{
		"data": records,
	})
}

// CreateRecord creates a new DNS record for a domain.
func (h *DNSHandler) CreateRecord(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	// Validate domain exists
	d, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	var req createDNSRecordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	record, err := h.DNSSvc.Create(domainID, req.Type, req.Name, req.Value, req.TTL, req.Priority)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": fmt.Sprintf("Failed to create DNS record: %v", err),
			},
		})
	}

	// Regenerate zone file (non-fatal)
	regenerateZone(h, domainID, d.Name)

	// Log activity
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "dns_create", "dns_record", record.ID, d.Name, c.IP())

	return c.Status(fiber.StatusCreated).JSON(record)
}

// UpdateRecord updates an existing DNS record.
func (h *DNSHandler) UpdateRecord(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid record ID",
			},
		})
	}

	// Get existing record to find domain_id
	existing, err := h.DNSSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "DNS record not found",
			},
		})
	}

	// Look up domain
	d, err := h.DomainSvc.GetByID(existing.DomainID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	var req updateDNSRecordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	record, err := h.DNSSvc.Update(id, req.Type, req.Name, req.Value, req.TTL, req.Priority)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": fmt.Sprintf("Failed to update DNS record: %v", err),
			},
		})
	}

	// Regenerate zone file (non-fatal)
	regenerateZone(h, existing.DomainID, d.Name)

	// Log activity
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "dns_update", "dns_record", record.ID, d.Name, c.IP())

	return c.JSON(record)
}

// DeleteRecord removes a DNS record.
func (h *DNSHandler) DeleteRecord(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid record ID",
			},
		})
	}

	// Get existing record to find domain_id
	existing, err := h.DNSSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "DNS record not found",
			},
		})
	}

	// Look up domain
	d, err := h.DomainSvc.GetByID(existing.DomainID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	if err := h.DNSSvc.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete DNS record",
			},
		})
	}

	// Regenerate zone file (non-fatal)
	regenerateZone(h, existing.DomainID, d.Name)

	// Log activity
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "dns_delete", "dns_record", existing.ID, d.Name, c.IP())

	return c.JSON(fiber.Map{"message": "DNS record deleted successfully"})
}

// ResetDefaults deletes all DNS records for a domain and recreates the default set.
func (h *DNSHandler) ResetDefaults(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	d, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	// Delete all existing records
	if err := h.DNSSvc.DeleteByDomain(domainID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete existing DNS records",
			},
		})
	}

	// Create default records
	serverIP := getServerIP()
	if err := h.DNSSvc.CreateDefaultRecords(domainID, d.Name, serverIP); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": fmt.Sprintf("Failed to create default DNS records: %v", err),
			},
		})
	}

	// Regenerate zone file
	regenerateZone(h, domainID, d.Name)

	// Log activity
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "dns_reset", "domain", domainID, d.Name, c.IP())

	// Return new records
	records, err := h.DNSSvc.ListByDomain(domainID)
	if err != nil {
		records = []dns.Record{}
	}

	return c.JSON(fiber.Map{
		"data":    records,
		"message": "DNS records reset to defaults",
	})
}

// regenerateZone rebuilds the BIND9 zone file for a domain and pushes it
// to the server via the agent. All errors are logged but non-fatal.
func regenerateZone(h *DNSHandler, domainID int64, domainName string) {
	// 1. List all records for the domain
	records, err := h.DNSSvc.ListByDomain(domainID)
	if err != nil {
		log.Printf("WARNING: failed to list DNS records for zone regeneration of %s: %v", domainName, err)
		return
	}

	// 2. Convert to ZoneRecord slice
	zoneRecords := make([]tmpl.ZoneRecord, 0, len(records))
	for _, r := range records {
		zr := tmpl.ZoneRecord{
			Name:  r.Name,
			TTL:   r.TTL,
			Class: "IN",
			Type:  r.Type,
			Value: r.Value,
		}

		// Handle MX/SRV priority
		if r.Priority != nil {
			zr.Priority = *r.Priority
		}

		zoneRecords = append(zoneRecords, zr)
	}

	// 3. Render zone file
	zoneContent, err := tmpl.RenderZoneFile(tmpl.ZoneFileData{
		Domain:  domainName,
		Records: zoneRecords,
	})
	if err != nil {
		log.Printf("WARNING: failed to render zone file for %s: %v", domainName, err)
		return
	}

	// 4. Write zone file via agent
	_, err = h.AgentClient.Call("dns_write_zone", map[string]interface{}{
		"domain":  domainName,
		"content": zoneContent,
	})
	if err != nil {
		log.Printf("WARNING: failed to write zone file for %s: %v", domainName, err)
		return
	}

	// 5. Ensure zone is registered in named.conf.local (idempotent)
	_, err = h.AgentClient.Call("dns_add_zone", map[string]interface{}{
		"domain": domainName,
	})
	if err != nil {
		log.Printf("WARNING: failed to register zone in BIND for %s: %v", domainName, err)
	}

	// 6. Reload DNS
	_, err = h.AgentClient.Call("dns_reload", nil)
	if err != nil {
		log.Printf("ERROR: dns reload failed after updating zone for %s: %v", domainName, err)
	}
}
