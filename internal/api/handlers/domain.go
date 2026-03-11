package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/php"
	"github.com/pinkpanel/pinkpanel/internal/core/subdomain"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// DomainHandler handles domain CRUD and lifecycle operations.
type DomainHandler struct {
	DB           *sql.DB
	DomainSvc    *domain.Service
	DNSSvc       *dns.Service
	SubdomainSvc *subdomain.Service
	AgentClient  *agent.Client
}

type createDomainRequest struct {
	Name       string `json:"name"`
	PHPVersion string `json:"php_version"`
	CreateWWW  bool   `json:"create_www"`
}

type updateDomainRequest struct {
	DocumentRoot string `json:"document_root"`
	PHPVersion   string `json:"php_version"`
}

// List returns a paginated list of domains.
func (h *DomainHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	domains, total, err := h.DomainSvc.List(
		c.Query("search"),
		c.Query("status"),
		page,
		perPage,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list domains",
			},
		})
	}

	if domains == nil {
		domains = []domain.Domain{}
	}

	return c.JSON(fiber.Map{
		"data":     domains,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// Get returns a single domain by ID.
func (h *DomainHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	d, err := h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	return c.JSON(d)
}

// Create provisions a new domain with NGINX configuration.
func (h *DomainHandler) Create(c *fiber.Ctx) error {
	var req createDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Domain name is required",
			},
		})
	}

	phpVersion := req.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.3"
	}

	// Create domain in DB
	d, err := h.DomainSvc.Create(req.Name, phpVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create domain",
			},
		})
	}

	// If create_www is true, adjust document root to use public_html
	docRoot := d.DocumentRoot
	if req.CreateWWW {
		docRoot = fmt.Sprintf("/var/www/%s/public_html", d.Name)
		d, err = h.DomainSvc.Update(d.ID, docRoot, d.PHPVersion)
		if err != nil {
			log.Printf("WARNING: failed to update document root for %s: %v", d.Name, err)
		}
	}

	// Create document root directory via agent
	_, err = h.AgentClient.Call("dir_create", map[string]interface{}{
		"path":  d.DocumentRoot,
		"owner": "www-data",
		"group": "www-data",
	})
	if err != nil {
		log.Printf("WARNING: failed to create document root for %s: %v", d.Name, err)
	}

	// Create default welcome page
	indexPath := fmt.Sprintf("%s/index.html", d.DocumentRoot)
	indexContent := tmpl.DefaultIndexPage(d.Name)
	if _, err := h.AgentClient.Call("file_write", map[string]interface{}{
		"path":    indexPath,
		"content": indexContent,
		"mode":    "0644",
	}); err != nil {
		log.Printf("WARNING: failed to create default index page for %s: %v", d.Name, err)
	}

	// Create PHP-FPM pool for this domain
	poolConfig := php.DefaultPoolConfig(d.Name, d.PHPVersion, nil)
	poolContent, err := tmpl.RenderPHPPool(tmpl.PHPPoolData{
		Domain:       poolConfig.Domain,
		User:         poolConfig.User,
		Group:        poolConfig.Group,
		ListenSocket: poolConfig.ListenSocket,
		PHPVersion:   poolConfig.PHPVersion,
		Settings:     poolConfig.Settings,
	})
	if err != nil {
		log.Printf("WARNING: failed to render PHP pool for %s: %v", d.Name, err)
	} else {
		if _, err := h.AgentClient.Call("php_write_pool", map[string]interface{}{
			"version": d.PHPVersion,
			"domain":  d.Name,
			"content": poolContent,
		}); err != nil {
			log.Printf("WARNING: failed to write PHP pool for %s: %v", d.Name, err)
		} else {
			if _, err := h.AgentClient.Call("php_reload", map[string]interface{}{
				"version": d.PHPVersion,
			}); err != nil {
				log.Printf("WARNING: failed to reload PHP-FPM after creating pool for %s: %v", d.Name, err)
			}
		}
	}

	// Create default DNS records (do this before NGINX so it always happens)
	serverIP := getServerIP()
	if err := h.DNSSvc.CreateDefaultRecords(d.ID, d.Name, serverIP); err != nil {
		log.Printf("WARNING: failed to create default DNS records for %s: %v", d.Name, err)
	} else {
		// Generate zone file and register in BIND
		provisionDNSZone(h.DNSSvc, h.AgentClient, d.ID, d.Name)
	}

	// Render NGINX vhost configuration
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       d.Name,
		DocumentRoot: d.DocumentRoot,
		PHPVersion:   d.PHPVersion,
	})
	if err != nil {
		log.Printf("WARNING: failed to render vhost for %s: %v", d.Name, err)
	} else {
		// Write vhost to sites-available
		configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    configPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write vhost config for %s: %v", d.Name, err)
		}

		// Write vhost to sites-enabled
		enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    enabledPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write sites-enabled config for %s: %v", d.Name, err)
		}

		// Test and reload NGINX
		_, err = h.AgentClient.Call("nginx_test", nil)
		if err != nil {
			log.Printf("WARNING: nginx config test failed after creating %s: %v", d.Name, err)
		}
		_, err = h.AgentClient.Call("nginx_reload", nil)
		if err != nil {
			log.Printf("ERROR: nginx reload failed after creating %s: %v", d.Name, err)
		}
	}

	// Log activity (non-critical)
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "domain_create", "domain", d.ID, d.Name, c.IP())

	return c.Status(fiber.StatusCreated).JSON(d)
}

// Update modifies an existing domain and re-renders its NGINX config.
func (h *DomainHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	existing, err := h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	var req updateDomainRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	docRoot := req.DocumentRoot
	if docRoot == "" {
		docRoot = existing.DocumentRoot
	}
	phpVersion := req.PHPVersion
	if phpVersion == "" {
		phpVersion = existing.PHPVersion
	}

	d, err := h.DomainSvc.Update(id, docRoot, phpVersion)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update domain",
			},
		})
	}

	// Re-render and write NGINX vhost
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       d.Name,
		DocumentRoot: d.DocumentRoot,
		PHPVersion:   d.PHPVersion,
	})
	if err != nil {
		log.Printf("WARNING: failed to render vhost for %s: %v", d.Name, err)
		return c.JSON(d)
	}

	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
	_, err = h.AgentClient.Call("file_write", map[string]interface{}{
		"path":    configPath,
		"content": vhostContent,
		"mode":    "0644",
	})
	if err != nil {
		log.Printf("WARNING: failed to write vhost config for %s: %v", d.Name, err)
		return c.JSON(d)
	}

	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
	_, err = h.AgentClient.Call("file_write", map[string]interface{}{
		"path":    enabledPath,
		"content": vhostContent,
		"mode":    "0644",
	})
	if err != nil {
		log.Printf("WARNING: failed to write sites-enabled config for %s: %v", d.Name, err)
	}

	_, err = h.AgentClient.Call("nginx_test", nil)
	if err != nil {
		log.Printf("WARNING: nginx config test failed after updating %s: %v", d.Name, err)
	}

	_, err = h.AgentClient.Call("nginx_reload", nil)
	if err != nil {
		log.Printf("ERROR: nginx reload failed after updating %s: %v", d.Name, err)
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "domain_update", "domain", d.ID, d.Name, c.IP())

	return c.JSON(d)
}

// Suspend suspends a domain, replacing its vhost with a suspended page.
func (h *DomainHandler) Suspend(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	d, err := h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	if err := h.DomainSvc.Suspend(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to suspend domain",
			},
		})
	}

	// Render suspended vhost
	vhostContent, err := tmpl.RenderNginxSuspended(tmpl.NginxSuspendedData{
		Domain: d.Name,
	})
	if err != nil {
		log.Printf("WARNING: failed to render suspended vhost for %s: %v", d.Name, err)
	} else {
		configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    configPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write suspended vhost for %s: %v", d.Name, err)
		}

		enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    enabledPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write sites-enabled suspended config for %s: %v", d.Name, err)
		}

		_, err = h.AgentClient.Call("nginx_reload", nil)
		if err != nil {
			log.Printf("ERROR: nginx reload failed after suspending %s: %v", d.Name, err)
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "domain_suspend", "domain", d.ID, d.Name, c.IP())

	// Re-fetch domain to get updated status
	d, err = h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get updated domain",
			},
		})
	}

	return c.JSON(d)
}

// Activate re-activates a suspended domain.
func (h *DomainHandler) Activate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	d, err := h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	if err := h.DomainSvc.Activate(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to activate domain",
			},
		})
	}

	// Re-render normal vhost
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       d.Name,
		DocumentRoot: d.DocumentRoot,
		PHPVersion:   d.PHPVersion,
	})
	if err != nil {
		log.Printf("WARNING: failed to render vhost for %s: %v", d.Name, err)
	} else {
		configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    configPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write vhost config for %s: %v", d.Name, err)
		}

		enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
		_, err = h.AgentClient.Call("file_write", map[string]interface{}{
			"path":    enabledPath,
			"content": vhostContent,
			"mode":    "0644",
		})
		if err != nil {
			log.Printf("WARNING: failed to write sites-enabled config for %s: %v", d.Name, err)
		}

		_, err = h.AgentClient.Call("nginx_reload", nil)
		if err != nil {
			log.Printf("ERROR: nginx reload failed after activating %s: %v", d.Name, err)
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "domain_activate", "domain", d.ID, d.Name, c.IP())

	// Re-fetch domain to get updated status
	d, err = h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get updated domain",
			},
		})
	}

	return c.JSON(d)
}

// Delete removes a domain, its NGINX config, and optionally its files.
func (h *DomainHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	d, err := h.DomainSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Domain not found",
			},
		})
	}

	// Clean up subdomain NGINX configs before DB cascade deletes the rows
	if h.SubdomainSvc != nil {
		subs, _ := h.SubdomainSvc.List(id)
		for _, sub := range subs {
			fqdn := fmt.Sprintf("%s.%s", sub.Name, d.Name)
			h.AgentClient.Call("file_delete", map[string]interface{}{"path": fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", fqdn)})
			h.AgentClient.Call("file_delete", map[string]interface{}{"path": fmt.Sprintf("/etc/nginx/sites-available/%s.conf", fqdn)})
		}
	}

	// Delete from DB (cascades to subdomains, dns_records)
	if err := h.DomainSvc.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete domain",
			},
		})
	}

	// Remove PHP-FPM pool
	poolPath := fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", d.PHPVersion, d.Name)
	if _, err := h.AgentClient.Call("file_delete", map[string]interface{}{"path": poolPath}); err != nil {
		log.Printf("WARNING: failed to delete PHP pool for %s: %v", d.Name, err)
	} else {
		h.AgentClient.Call("php_reload", map[string]interface{}{"version": d.PHPVersion})
	}

	// Remove NGINX config files
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
	_, err = h.AgentClient.Call("file_delete", map[string]interface{}{
		"path": configPath,
	})
	if err != nil {
		log.Printf("WARNING: failed to delete sites-available config for %s: %v", d.Name, err)
	}

	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
	_, err = h.AgentClient.Call("file_delete", map[string]interface{}{
		"path": enabledPath,
	})
	if err != nil {
		log.Printf("WARNING: failed to delete sites-enabled config for %s: %v", d.Name, err)
	}

	// Reload NGINX
	_, err = h.AgentClient.Call("nginx_reload", nil)
	if err != nil {
		log.Printf("ERROR: nginx reload failed after deleting %s: %v", d.Name, err)
	}

	// Remove DNS records and zone
	if err := h.DNSSvc.DeleteByDomain(id); err != nil {
		log.Printf("WARNING: failed to delete DNS records for %s: %v", d.Name, err)
	}
	_, err = h.AgentClient.Call("dns_remove_zone", map[string]interface{}{
		"domain": d.Name,
	})
	if err != nil {
		log.Printf("WARNING: failed to remove DNS zone for %s: %v", d.Name, err)
	}
	_, _ = h.AgentClient.Call("dns_reload", nil)

	// Optionally remove document root
	removeFiles := c.Query("remove_files") == "true"
	if removeFiles {
		_, err = h.AgentClient.Call("file_delete", map[string]interface{}{
			"path":      d.DocumentRoot,
			"recursive": true,
		})
		if err != nil {
			log.Printf("WARNING: failed to remove document root for %s: %v", d.Name, err)
		}
	}

	// Log activity (non-critical)
	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "domain_delete", "domain", d.ID, d.Name, c.IP())

	return c.JSON(fiber.Map{"message": "Domain deleted successfully"})
}

// getServerIP returns the server's primary public IPv4 address.
// Skips loopback and private Docker/container ranges.
func getServerIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("WARNING: failed to get interface addresses: %v", err)
		return "127.0.0.1"
	}

	var fallback string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		ip := ipNet.IP.String()
		// Prefer non-private IPs (public)
		if !ipNet.IP.IsPrivate() {
			return ip
		}
		// Keep first private IP as fallback
		if fallback == "" {
			fallback = ip
		}
	}
	if fallback != "" {
		return fallback
	}
	log.Printf("WARNING: no non-loopback IPv4 found, using 127.0.0.1 for DNS records")
	return "127.0.0.1"
}

// provisionDNSZone generates a BIND zone file from DNS records and registers
// it in named.conf.local. All errors are logged but non-fatal.
func provisionDNSZone(dnsSvc *dns.Service, agentClient *agent.Client, domainID int64, domainName string) {
	records, err := dnsSvc.ListByDomain(domainID)
	if err != nil {
		log.Printf("WARNING: failed to list DNS records for zone provisioning of %s: %v", domainName, err)
		return
	}

	zoneRecords := make([]tmpl.ZoneRecord, 0, len(records))
	for _, r := range records {
		zr := tmpl.ZoneRecord{
			Name:  r.Name,
			TTL:   r.TTL,
			Class: "IN",
			Type:  r.Type,
			Value: r.Value,
		}
		if r.Priority != nil {
			zr.Priority = *r.Priority
		}
		zoneRecords = append(zoneRecords, zr)
	}

	zoneContent, err := tmpl.RenderZoneFile(tmpl.ZoneFileData{
		Domain:  domainName,
		Records: zoneRecords,
	})
	if err != nil {
		log.Printf("WARNING: failed to render zone file for %s: %v", domainName, err)
		return
	}

	// Write zone file
	_, err = agentClient.Call("dns_write_zone", map[string]interface{}{
		"domain":  domainName,
		"content": zoneContent,
	})
	if err != nil {
		log.Printf("WARNING: failed to write zone file for %s: %v", domainName, err)
		return
	}

	// Register zone in BIND config
	_, err = agentClient.Call("dns_add_zone", map[string]interface{}{
		"domain": domainName,
	})
	if err != nil {
		log.Printf("WARNING: failed to add zone to BIND for %s: %v", domainName, err)
		return
	}

	// Reload DNS
	_, err = agentClient.Call("dns_reload", nil)
	if err != nil {
		log.Printf("ERROR: dns reload failed after provisioning zone for %s: %v", domainName, err)
	}
}
