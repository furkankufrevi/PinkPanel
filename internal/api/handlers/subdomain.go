package handlers

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/subdomain"
	"github.com/pinkpanel/pinkpanel/internal/db"
	"github.com/pinkpanel/pinkpanel/internal/template"
)

type SubdomainHandler struct {
	DB           *sql.DB
	SubdomainSvc *subdomain.Service
	DomainSvc    *domain.Service
	AgentClient  *agent.Client
}

// List returns subdomains for a domain.
func (h *SubdomainHandler) List(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	subs, err := h.SubdomainSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if subs == nil {
		subs = []subdomain.Subdomain{}
	}
	return c.JSON(fiber.Map{"data": subs})
}

// Create creates a subdomain.
func (h *SubdomainHandler) Create(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "name is required"}})
	}

	sub, err := h.SubdomainSvc.Create(domainID, req.Name, dom.Name)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Create document root via agent
	if _, err := h.AgentClient.Call("dir_create", map[string]any{
		"path": sub.DocumentRoot,
		"mode": "0755",
	}); err != nil {
		log.Error().Err(err).Msg("failed to create subdomain document root")
	}

	// Generate NGINX vhost for the subdomain
	fqdn := fmt.Sprintf("%s.%s", req.Name, dom.Name)
	vhostConfig, err := template.RenderNginxVhost(template.NginxVhostData{
		Domain:       fqdn,
		DocumentRoot: sub.DocumentRoot,
		PHPVersion:   dom.PHPVersion,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render subdomain vhost")
	} else {
		vhostPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", fqdn)
		enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", fqdn)

		// Write to sites-available
		if _, err := h.AgentClient.Call("file_write", map[string]any{
			"path": vhostPath, "content": vhostConfig, "mode": "0644",
		}); err != nil {
			log.Error().Err(err).Msg("failed to write subdomain vhost")
		}

		// Write to sites-enabled (same content, avoids needing symlink command)
		if _, err := h.AgentClient.Call("file_write", map[string]any{
			"path": enabledPath, "content": vhostConfig, "mode": "0644",
		}); err != nil {
			log.Error().Err(err).Msg("failed to enable subdomain vhost")
		}

		// Reload NGINX
		if _, err := h.AgentClient.Call("service_control", map[string]any{
			"service": "nginx", "action": "reload",
		}); err != nil {
			log.Error().Err(err).Msg("failed to reload nginx")
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_subdomain", "subdomain", sub.ID, fqdn, c.IP())

	return c.Status(201).JSON(sub)
}

// Delete removes a subdomain.
func (h *SubdomainHandler) Delete(c *fiber.Ctx) error {
	subID, err := strconv.ParseInt(c.Params("subId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid subdomain ID"}})
	}

	sub, err := h.SubdomainSvc.Delete(subID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Get parent domain name for FQDN
	dom, _ := h.DomainSvc.GetByID(sub.DomainID)
	var fqdn string
	if dom != nil {
		fqdn = fmt.Sprintf("%s.%s", sub.Name, dom.Name)
	} else {
		fqdn = sub.Name
	}

	// Remove NGINX vhost
	vhostPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", fqdn)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", fqdn)

	if _, err := h.AgentClient.Call("file_delete", map[string]any{"path": enabledPath}); err != nil {
		log.Error().Err(err).Msg("failed to remove subdomain enabled vhost")
	}
	if _, err := h.AgentClient.Call("file_delete", map[string]any{"path": vhostPath}); err != nil {
		log.Error().Err(err).Msg("failed to remove subdomain vhost")
	}

	// Reload NGINX
	if _, err := h.AgentClient.Call("service_control", map[string]any{
		"service": "nginx", "action": "reload",
	}); err != nil {
		log.Error().Err(err).Msg("failed to reload nginx")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_subdomain", "subdomain", subID, fqdn, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}
