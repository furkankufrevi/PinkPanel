package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/ssl"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

type SSLHandler struct {
	DB          *sql.DB
	SSLSvc      *ssl.Service
	DomainSvc   *domain.Service
	DNSSvc      *dns.Service
	AgentClient *agent.Client
	ACMESvc     *ssl.ACMEService
}

// securedComponent represents a single component's SSL status.
type securedComponent struct {
	Name    string `json:"name"`
	Secured bool   `json:"secured"`
}

// GetCertificate returns the SSL certificate for a domain with secured components.
func (h *SSLHandler) GetCertificate(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	cert, err := h.SSLSvc.GetByDomainID(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if cert == nil {
		return c.JSON(fiber.Map{"installed": false})
	}

	// Build secured components by parsing the stored domains field
	securedDomains := make(map[string]bool)
	if cert.Domains != nil && *cert.Domains != "" {
		for _, d := range strings.Split(*cert.Domains, ",") {
			securedDomains[strings.TrimSpace(d)] = true
		}
	}

	components := fiber.Map{
		"domain":        securedComponent{Name: dom.Name, Secured: securedDomains[dom.Name]},
		"www":           securedComponent{Name: "www." + dom.Name, Secured: securedDomains["www."+dom.Name]},
		"mail":          securedComponent{Name: "mail." + dom.Name, Secured: securedDomains["mail."+dom.Name]},
		"webmail":       securedComponent{Name: "webmail." + dom.Name, Secured: securedDomains["webmail."+dom.Name]},
		"wildcard":      securedComponent{Name: "*." + dom.Name, Secured: securedDomains["*."+dom.Name]},
		"mail_services": securedComponent{Name: "IMAP, POP, SMTP", Secured: cert.MailSSL},
	}

	return c.JSON(fiber.Map{
		"installed":          true,
		"id":                 cert.ID,
		"type":               cert.Type,
		"issuer":             cert.Issuer,
		"domains":            cert.Domains,
		"issued_at":          cert.IssuedAt,
		"expires_at":         cert.ExpiresAt,
		"auto_renew":         cert.AutoRenew,
		"force_https":        cert.ForceHTTPS,
		"hsts":               cert.HSTS,
		"mail_ssl":           cert.MailSSL,
		"challenge_type":     cert.ChallengeType,
		"created_at":         cert.CreatedAt,
		"secured_components": components,
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
	cert, err := h.SSLSvc.Install(domainID, "custom", certPath, keyPath, chainPath, "", dom.Name, expiresAt, req.ForceHTTPS)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Update NGINX vhost with SSL (HSTS off by default)
	h.updateNginxWithSSL(dom, certPath, keyPath, chainPath, req.ForceHTTPS, false)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "install_ssl", "domain", domainID, fmt.Sprintf("custom SSL for %s", dom.Name), c.IP())

	return c.Status(201).JSON(fiber.Map{
		"installed":  true,
		"id":         cert.ID,
		"type":       cert.Type,
		"expires_at": cert.ExpiresAt,
	})
}

// IssueLetsEncrypt obtains a Let's Encrypt certificate with multi-SAN support.
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

	var req struct {
		SecureDomain   *bool `json:"secure_domain"`
		SecureWildcard bool  `json:"secure_wildcard"`
		IncludeWWW     *bool `json:"include_www"`
		SecureWebmail  bool  `json:"secure_webmail"`
		SecureMail     bool  `json:"secure_mail"`
		AssignToMail   bool  `json:"assign_to_mail"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	// Default: secure_domain=true, include_www=true if not specified
	secureDomain := true
	if req.SecureDomain != nil {
		secureDomain = *req.SecureDomain
	}
	includeWWW := true
	if req.IncludeWWW != nil {
		includeWWW = *req.IncludeWWW
	}

	// Build domain list from booleans
	var domains []string
	if secureDomain {
		domains = append(domains, dom.Name)
	}
	if req.SecureWildcard {
		domains = append(domains, "*."+dom.Name)
	}
	if includeWWW && !req.SecureWildcard {
		// www is covered by wildcard, only add explicitly if no wildcard
		domains = append(domains, "www."+dom.Name)
	}
	if req.SecureWebmail && !req.SecureWildcard {
		domains = append(domains, "webmail."+dom.Name)
	}
	if req.SecureMail && !req.SecureWildcard {
		domains = append(domains, "mail."+dom.Name)
	}

	if len(domains) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "at least one domain must be selected"}})
	}

	// Determine challenge type: wildcard requires DNS-01
	challengeType := "http-01"
	var issued *ssl.IssuedCert
	if req.SecureWildcard {
		challengeType = "dns-01"
		issued, err = h.ACMESvc.IssueCertificateDNS01(domains, domainID, dom.Name)
	} else {
		issued, err = h.ACMESvc.IssueCertificate(domains, dom.DocumentRoot)
	}
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

	// Store in database with new fields
	cert, err := h.SSLSvc.Install(domainID, "letsencrypt", certPath, keyPath, chainPath, issued.Issuer, issued.Domains, issued.ExpiresAt, true, ssl.InstallOpts{
		HSTS:          false,
		MailSSL:       req.AssignToMail,
		ChallengeType: challengeType,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// If assign_to_mail, configure Postfix/Dovecot with the cert
	if req.AssignToMail {
		if _, err := h.AgentClient.Call("email_configure_ssl", map[string]any{
			"domain":    dom.Name,
			"cert_path": certPath,
			"key_path":  keyPath,
		}); err != nil {
			log.Error().Err(err).Str("domain", dom.Name).Msg("failed to assign SSL to mail services")
		}
	}

	// Update NGINX with SSL (HSTS off by default, user can enable via toggle)
	h.updateNginxWithSSL(dom, certPath, keyPath, chainPath, true, false)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "issue_ssl", "domain", domainID, fmt.Sprintf("Let's Encrypt for %s", issued.Domains), c.IP())

	return c.Status(201).JSON(fiber.Map{
		"installed":      true,
		"id":             cert.ID,
		"type":           "letsencrypt",
		"issuer":         issued.Issuer,
		"domains":        issued.Domains,
		"expires_at":     issued.ExpiresAt,
		"auto_renew":     true,
		"hsts":           cert.HSTS,
		"mail_ssl":       cert.MailSSL,
		"challenge_type": challengeType,
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
		enableVhost(h.AgentClient, dom.Name)
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

// ToggleForceHTTPS enables or disables HTTP→HTTPS redirect.
func (h *SSLHandler) ToggleForceHTTPS(c *fiber.Ctx) error {
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

	cert, err := h.SSLSvc.GetByDomainID(domainID)
	if err != nil || cert == nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "no SSL certificate installed"}})
	}

	if err := h.SSLSvc.ToggleForceHTTPS(domainID, req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Re-render NGINX config with updated force_https setting
	chainPath := ""
	if cert.ChainPath != nil {
		chainPath = *cert.ChainPath
	}
	h.updateNginxWithSSL(dom, cert.CertPath, cert.KeyPath, chainPath, req.Enabled, cert.HSTS)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "toggle_ssl_force_https", "domain", domainID, fmt.Sprintf("force_https=%v", req.Enabled), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "force_https": req.Enabled})
}

// ToggleHSTS enables or disables HSTS for a domain.
func (h *SSLHandler) ToggleHSTS(c *fiber.Ctx) error {
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

	cert, err := h.SSLSvc.GetByDomainID(domainID)
	if err != nil || cert == nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "no SSL certificate installed"}})
	}

	if err := h.SSLSvc.ToggleHSTS(domainID, req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	// Re-render NGINX config with updated HSTS setting
	chainPath := ""
	if cert.ChainPath != nil {
		chainPath = *cert.ChainPath
	}
	h.updateNginxWithSSL(dom, cert.CertPath, cert.KeyPath, chainPath, cert.ForceHTTPS, req.Enabled)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "toggle_ssl_hsts", "domain", domainID, fmt.Sprintf("hsts=%v", req.Enabled), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "hsts": req.Enabled})
}

func (h *SSLHandler) updateNginxWithSSL(dom *domain.Domain, certPath, keyPath, chainPath string, forceHTTPS, hsts bool) {
	hstsMaxAge := 0
	if hsts {
		hstsMaxAge = 31536000
	}
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
		HSTS:         hsts,
		HSTSMaxAge:   hstsMaxAge,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to render nginx vhost with SSL")
		return
	}
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", dom.Name)
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": configPath, "content": vhostContent, "mode": "0644",
	}); err != nil {
		log.Error().Err(err).Msg("failed to write nginx sites-available config")
		return
	}
	enableVhost(h.AgentClient, dom.Name)
	if _, err := h.AgentClient.Call("nginx_test", nil); err != nil {
		log.Error().Err(err).Msg("nginx config test failed")
		return
	}
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload nginx")
	}
}
