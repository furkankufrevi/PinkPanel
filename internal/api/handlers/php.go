package handlers

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/php"
	"github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

type PHPHandler struct {
	DB          *sql.DB
	PHPSvc      *php.Service
	DomainSvc   *domain.Service
	SSLSvc      *ssl.Service
	AgentClient *agent.Client
}

// GetVersions returns available PHP versions.
func (h *PHPHandler) GetVersions(c *fiber.Ctx) error {
	versions := h.PHPSvc.ListInstalledVersions()
	return c.JSON(fiber.Map{"data": versions})
}

// GetDomainPHP returns PHP settings for a domain.
func (h *PHPHandler) GetDomainPHP(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}
	settings, err := h.PHPSvc.GetDomainPHP(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(settings)
}

// UpdateDomainPHP updates PHP version and settings, then applies config via agent.
func (h *PHPHandler) UpdateDomainPHP(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Version  string            `json:"version"`
		Settings map[string]string `json:"settings"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	// Get domain info for template rendering
	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, dom) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	// Update in database
	settings, err := h.PHPSvc.UpdateDomainPHP(domainID, req.Version, req.Settings)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Render and write PHP-FPM pool config via agent
	poolConfig := php.DefaultPoolConfig(dom.Name, req.Version, req.Settings)
	poolContent, err := tmpl.RenderPHPPool(tmpl.PHPPoolData{
		Domain:       poolConfig.Domain,
		User:         poolConfig.User,
		Group:        poolConfig.Group,
		ListenSocket: poolConfig.ListenSocket,
		PHPVersion:   poolConfig.PHPVersion,
		Settings:     poolConfig.Settings,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render PHP pool config")
	} else {
		if _, err := h.AgentClient.Call("php_write_pool", map[string]any{
			"version": req.Version,
			"domain":  dom.Name,
			"content": poolContent,
		}); err != nil {
			log.Error().Err(err).Msg("failed to write PHP pool config via agent")
		} else {
			// Reload PHP-FPM
			if _, err := h.AgentClient.Call("php_reload", map[string]any{
				"version": req.Version,
			}); err != nil {
				log.Error().Err(err).Msg("failed to reload PHP-FPM")
			}
		}
	}

	// Update NGINX vhost to point to new PHP-FPM socket (preserve SSL if present)
	vhostData := tmpl.NginxVhostData{
		Domain:       dom.Name,
		DocumentRoot: dom.DocumentRoot,
		PHPVersion:   req.Version,
	}
	if cert, err := h.SSLSvc.GetByDomainID(domainID); err == nil && cert != nil {
		vhostData.SSLEnabled = true
		vhostData.SSLCertPath = cert.CertPath
		vhostData.SSLKeyPath = cert.KeyPath
		if cert.ChainPath != nil {
			vhostData.SSLChainPath = *cert.ChainPath
		}
		vhostData.ForceHTTPS = cert.ForceHTTPS
		vhostData.HTTP2 = true
		vhostData.HSTS = true
		vhostData.HSTSMaxAge = 31536000
	}
	vhostContent, err := tmpl.RenderNginxVhost(vhostData)
	if err != nil {
		log.Error().Err(err).Msg("failed to render nginx vhost")
	} else {
		configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", dom.Name)
		if _, err := h.AgentClient.Call("file_write", map[string]any{
			"path":    configPath,
			"content": vhostContent,
			"mode":    "0644",
		}); err != nil {
			log.Error().Err(err).Msg("failed to write nginx vhost config")
		}
		enableVhost(h.AgentClient, dom.Name)
		if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
			log.Error().Err(err).Msg("failed to reload nginx")
		}
	}

	// Log activity
	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_php", "domain", domainID, fmt.Sprintf("PHP %s", req.Version), c.IP())

	return c.JSON(settings)
}

// GetPHPInfo returns PHP configuration info for a domain.
func (h *PHPHandler) GetPHPInfo(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, dom) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	resp, err := h.AgentClient.Call("php_info", map[string]any{
		"version": dom.PHPVersion,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to get PHP info: " + err.Error()}})
	}

	return c.JSON(fiber.Map{
		"version": dom.PHPVersion,
		"info":    resp.Result,
	})
}
