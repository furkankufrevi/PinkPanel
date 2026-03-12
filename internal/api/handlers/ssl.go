package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

type SSLHandler struct {
	DB          *sql.DB
	SSLSvc      *ssl.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
	ACMESvc     *ssl.ACMEService
}

// GetCertificate returns the SSL certificate for a domain.
func (h *SSLHandler) GetCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	cert, err := h.SSLSvc.GetByDomainID(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if cert == nil {
		return c.JSON(fiber.Map{"installed": false})
	}

	return c.JSON(fiber.Map{
		"installed":  true,
		"id":         cert.ID,
		"type":       cert.Type,
		"issuer":     cert.Issuer,
		"domains":    cert.Domains,
		"issued_at":  cert.IssuedAt,
		"expires_at": cert.ExpiresAt,
		"auto_renew": cert.AutoRenew,
		"created_at": cert.CreatedAt,
	})
}

// InstallCertificate installs a custom SSL certificate.
func (h *SSLHandler) InstallCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Certificate string `json:"certificate"`
		PrivateKey  string `json:"private_key"`
		Chain       string `json:"chain"`
		ForceHTTPS  bool   `json:"force_https"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if req.Certificate == "" || req.PrivateKey == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "certificate and private_key are required"}})
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

	// Write certificate files via agent
	resp, err := h.AgentClient.Call("ssl_write_cert", map[string]any{
		"domain": dom.Name,
		"cert":   req.Certificate,
		"key":    req.PrivateKey,
		"chain":  req.Chain,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to write SSL certificate: " + err.Error()}})
	}

	// Extract paths from agent response
	result, _ := resp.Result.(map[string]interface{})
	certPath, _ := result["cert_path"].(string)
	keyPath, _ := result["key_path"].(string)
	chainPath := ""
	if cp, ok := result["chain_path"].(string); ok {
		chainPath = cp
	}

	// Store in database — expires_at set to 1 year for custom certs (user should update)
	expiresAt := time.Now().AddDate(1, 0, 0)
	cert, err := h.SSLSvc.Install(domainID, "custom", certPath, keyPath, chainPath, "", dom.Name, expiresAt)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Update NGINX vhost with SSL
	h.updateNginxWithSSL(dom, certPath, keyPath, chainPath, req.ForceHTTPS)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "install_ssl", "domain", domainID, fmt.Sprintf("custom SSL for %s", dom.Name), c.IP())

	return c.Status(201).JSON(fiber.Map{
		"installed":  true,
		"id":         cert.ID,
		"type":       cert.Type,
		"expires_at": cert.ExpiresAt,
	})
}

// IssueLetsEncrypt obtains a Let's Encrypt certificate via ACME HTTP-01.
func (h *SSLHandler) IssueLetsEncrypt(c *fiber.Ctx) error {
	if h.ACMESvc == nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "not_configured", "message": "ACME service is not configured — set admin email in settings"}})
	}

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

	// Build domain list: primary + www
	domains := []string{dom.Name}
	var req struct {
		IncludeWWW bool `json:"include_www"`
	}
	if err := c.BodyParser(&req); err == nil && req.IncludeWWW {
		domains = append(domains, "www."+dom.Name)
	}

	// Issue certificate (challenge tokens written via agent in the provider)
	issued, err := h.ACMESvc.IssueCertificate(domains, dom.DocumentRoot)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "acme_error", "message": "Let's Encrypt issuance failed: " + err.Error()}})
	}

	// Write cert files via agent
	resp, err := h.AgentClient.Call("ssl_write_cert", map[string]any{
		"domain": dom.Name,
		"cert":   issued.Certificate,
		"key":    issued.PrivateKey,
		"chain":  issued.IssuerCert,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to write certificate files: " + err.Error()}})
	}

	result, _ := resp.Result.(map[string]interface{})
	certPath, _ := result["cert_path"].(string)
	keyPath, _ := result["key_path"].(string)
	chainPath := ""
	if cp, ok := result["chain_path"].(string); ok {
		chainPath = cp
	}

	// Store in database
	cert, err := h.SSLSvc.Install(domainID, "letsencrypt", certPath, keyPath, chainPath, issued.Issuer, issued.Domains, issued.ExpiresAt)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Update NGINX with SSL
	h.updateNginxWithSSL(dom, certPath, keyPath, chainPath, true)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "issue_ssl", "domain", domainID, fmt.Sprintf("Let's Encrypt for %s", issued.Domains), c.IP())

	return c.Status(201).JSON(fiber.Map{
		"installed":  true,
		"id":         cert.ID,
		"type":       "letsencrypt",
		"issuer":     issued.Issuer,
		"domains":    issued.Domains,
		"expires_at": issued.ExpiresAt,
		"auto_renew": true,
	})
}

// DeleteCertificate removes the SSL certificate for a domain.
func (h *SSLHandler) DeleteCertificate(c *fiber.Ctx) error {
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

	// Delete certificate files via agent
	if _, err := h.AgentClient.Call("ssl_delete_cert", map[string]any{
		"domain": dom.Name,
	}); err != nil {
		log.Error().Err(err).Str("domain", dom.Name).Msg("failed to delete SSL files via agent")
	}

	// Remove from database
	if err := h.SSLSvc.Delete(domainID); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Update NGINX vhost to remove SSL
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       dom.Name,
		DocumentRoot: dom.DocumentRoot,
		PHPVersion:   dom.PHPVersion,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render nginx vhost")
	} else {
		configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", dom.Name)
		if _, err := h.AgentClient.Call("file_write", map[string]any{
			"path": configPath, "content": vhostContent, "mode": "0644",
		}); err != nil {
			log.Error().Err(err).Msg("failed to write nginx config")
		}
		if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
			log.Error().Err(err).Msg("failed to reload nginx")
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_ssl", "domain", domainID, dom.Name, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// ToggleAutoRenew enables or disables auto-renewal.
func (h *SSLHandler) ToggleAutoRenew(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if err := h.SSLSvc.ToggleAutoRenew(domainID, req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "toggle_ssl_autorenew", "domain", domainID, fmt.Sprintf("auto_renew=%v", req.Enabled), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "auto_renew": req.Enabled})
}

func (h *SSLHandler) updateNginxWithSSL(dom *domain.Domain, certPath, keyPath, chainPath string, forceHTTPS bool) {
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       dom.Name,
		DocumentRoot: dom.DocumentRoot,
		PHPVersion:   dom.PHPVersion,
		SSLEnabled:   true,
		SSLCertPath:  certPath,
		SSLKeyPath:   keyPath,
		SSLChainPath: chainPath,
		ForceHTTPS:   forceHTTPS,
		HTTP2:        true,
		HSTS:         true,
		HSTSMaxAge:   31536000,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render nginx vhost with SSL")
		return
	}
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", dom.Name)
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": configPath, "content": vhostContent, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Msg("failed to write nginx config")
		return
	}
	if _, err := h.AgentClient.Call("nginx_test", nil); err != nil {
		log.Error().Err(err).Msg("nginx config test failed")
		return
	}
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload nginx")
	}
}
