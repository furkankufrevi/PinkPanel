package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/api/middleware"
	"github.com/pinkpanel/pinkpanel/internal/core/dns"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/php"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// DomainHandler handles domain CRUD and lifecycle operations.
type DomainHandler struct {
	DB          *sql.DB
	DomainSvc   *domain.Service
	DNSSvc      *dns.Service
	AgentClient *agent.Client
}

type createDomainRequest struct {
	Name      string `json:"name"`
	PHPVersion string `json:"php_version"`
	CreateWWW bool   `json:"create_www"`
	ParentID  *int64 `json:"parent_id"`
}

type updateDomainRequest struct {
	DocumentRoot string `json:"document_root"`
	PHPVersion   string `json:"php_version"`
	SeparateDNS  *bool  `json:"separate_dns"`
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

	// Super admins see all domains; others see only their own
	var filterAdminID int64
	if !middleware.IsSuperAdmin(c) {
		filterAdminID, _ = c.Locals("admin_id").(int64)
	}

	domains, total, err := h.DomainSvc.List(
		c.Query("search"),
		c.Query("status"),
		page,
		perPage,
		filterAdminID,
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

	if !checkDomainAccess(c, d) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
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

	// If creating a subdomain, build the FQDN
	var parentDomain *domain.Domain
	if req.ParentID != nil {
		var err error
		parentDomain, err = h.DomainSvc.GetByID(*req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "VALIDATION_ERROR",
					"message": "Parent domain not found",
				},
			})
		}
		if parentDomain.ParentID != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "VALIDATION_ERROR",
					"message": "Cannot create subdomain of a subdomain",
				},
			})
		}
		// Build FQDN: req.Name should be just the subdomain prefix (e.g. "blog")
		// or already FQDN — if it doesn't end with parent name, prepend
		if !containsSuffix(req.Name, parentDomain.Name) {
			req.Name = req.Name + "." + parentDomain.Name
		}
	}

	// Create domain in DB with user-scoped document root
	adminID, _ := c.Locals("admin_id").(int64)
	var systemUsername string
	h.DB.QueryRow("SELECT COALESCE(system_username, 'www-data') FROM admins WHERE id = ?", adminID).Scan(&systemUsername)
	d, err := h.DomainSvc.CreateForUser(req.Name, phpVersion, req.ParentID, adminID, systemUsername)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": fmt.Sprintf("Failed to create domain: %v", err),
			},
		})
	}

	// Create document root directory via agent (owned by system user)
	fileOwner := systemUsername
	if fileOwner == "" {
		fileOwner = "www-data"
	}
	_, err = h.AgentClient.Call("dir_create", map[string]interface{}{
		"path":  d.DocumentRoot,
		"owner": fileOwner,
		"group": fileOwner,
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

	// Create PHP-FPM pool for this domain (runs as system user)
	poolConfig := php.DefaultPoolConfig(d.Name, d.PHPVersion, nil)
	poolConfig.User = fileOwner
	poolConfig.Group = fileOwner
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

	// DNS handling
	serverIP := getServerIP()
	serverIPv6 := getServerIPv6()
	if parentDomain != nil {
		// Subdomain: add A record to parent zone (separate_dns defaults to false)
		subPrefix := extractSubPrefix(d.Name, parentDomain.Name)
		if _, err := h.DNSSvc.Create(parentDomain.ID, "A", subPrefix, serverIP, 3600, nil); err != nil {
			log.Printf("WARNING: failed to create DNS A record for subdomain %s: %v", d.Name, err)
		} else {
			provisionDNSZone(h.DNSSvc, h.AgentClient, parentDomain.ID, parentDomain.Name)
		}
		// Add AAAA record for subdomain if IPv6 available
		if serverIPv6 != "" {
			h.DNSSvc.Create(parentDomain.ID, "AAAA", subPrefix, serverIPv6, 3600, nil)
		}
		// Clean up any stale separate zone for this subdomain (from previous creation/toggle)
		h.AgentClient.Call("dns_remove_zone", map[string]interface{}{"domain": d.Name})
		h.AgentClient.Call("dns_reload", nil)
	} else {
		// Root domain: create full DNS zone
		if err := h.DNSSvc.CreateDefaultRecords(d.ID, d.Name, serverIP, serverIPv6); err != nil {
			log.Printf("WARNING: failed to create default DNS records for %s: %v", d.Name, err)
		} else {
			provisionDNSZone(h.DNSSvc, h.AgentClient, d.ID, d.Name)
		}
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
	action := "domain_create"
	if parentDomain != nil {
		action = "subdomain_create"
	}
	_ = db.LogActivity(h.DB, adminID, action, "domain", d.ID, d.Name, c.IP())

	// Run system health checks and collect warnings for the response
	warnings := h.checkSystemHealth()

	if len(warnings) > 0 {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"data":     d,
			"warnings": warnings,
		})
	}
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

	if !checkDomainAccess(c, existing) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
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

	// Handle separate_dns toggle for subdomains
	if req.SeparateDNS != nil && existing.ParentID != nil {
		newVal := *req.SeparateDNS
		if newVal != existing.SeparateDNS {
			h.toggleSeparateDNS(existing, newVal)
		}
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

// toggleSeparateDNS handles the DNS transition when toggling separate_dns.
func (h *DomainHandler) toggleSeparateDNS(d *domain.Domain, separateDNS bool) {
	parent, err := h.DomainSvc.GetByID(*d.ParentID)
	if err != nil {
		log.Printf("WARNING: failed to get parent domain for DNS toggle: %v", err)
		return
	}
	subPrefix := extractSubPrefix(d.Name, parent.Name)
	serverIP := getServerIP()
	serverIPv6 := getServerIPv6()

	if separateDNS {
		// Turning ON: remove A from parent zone, create own zone
		_ = h.DNSSvc.DeleteByName(parent.ID, subPrefix)
		provisionDNSZone(h.DNSSvc, h.AgentClient, parent.ID, parent.Name)

		if err := h.DNSSvc.CreateDefaultRecords(d.ID, d.Name, serverIP, serverIPv6); err != nil {
			log.Printf("WARNING: failed to create DNS zone for subdomain %s: %v", d.Name, err)
		} else {
			provisionDNSZone(h.DNSSvc, h.AgentClient, d.ID, d.Name)
		}
	} else {
		// Turning OFF: remove subdomain zone, add A back to parent
		_ = h.DNSSvc.DeleteByDomain(d.ID)
		if _, err := h.AgentClient.Call("dns_remove_zone", map[string]interface{}{"domain": d.Name}); err != nil {
			log.Printf("WARNING: failed to remove DNS zone for %s: %v", d.Name, err)
		}
		if _, err := h.AgentClient.Call("dns_reload", nil); err != nil {
			log.Printf("ERROR: dns reload failed after removing zone for %s: %v", d.Name, err)
		}

		if _, err := h.DNSSvc.Create(parent.ID, "A", subPrefix, serverIP, 3600, nil); err != nil {
			log.Printf("WARNING: failed to re-add DNS A record for %s: %v", d.Name, err)
		} else {
			provisionDNSZone(h.DNSSvc, h.AgentClient, parent.ID, parent.Name)
		}
	}

	h.DomainSvc.UpdateSeparateDNS(d.ID, separateDNS)
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

	if !checkDomainAccess(c, d) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
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

	if !checkDomainAccess(c, d) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
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

	if !checkDomainAccess(c, d) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	// Clean up child subdomain resources before DB cascade deletes them
	children, _ := h.DomainSvc.GetChildren(id)
	for _, child := range children {
		h.AgentClient.Call("file_delete", map[string]interface{}{"path": fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", child.Name)})
		h.AgentClient.Call("file_delete", map[string]interface{}{"path": fmt.Sprintf("/etc/nginx/sites-available/%s.conf", child.Name)})
		// Remove child PHP pool
		poolPath := fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", child.PHPVersion, child.Name)
		h.AgentClient.Call("file_delete", map[string]interface{}{"path": poolPath})
		// Remove child SSL certs
		h.AgentClient.Call("ssl_delete_cert", map[string]interface{}{"domain": child.Name})
		// Remove child DNS zone if it has separate DNS
		if child.SeparateDNS {
			h.AgentClient.Call("dns_remove_zone", map[string]interface{}{"domain": child.Name})
		}
	}

	// If this is a subdomain with non-separate DNS, clean the A record from parent
	if d.ParentID != nil && !d.SeparateDNS {
		parent, err := h.DomainSvc.GetByID(*d.ParentID)
		if err == nil {
			subPrefix := extractSubPrefix(d.Name, parent.Name)
			_ = h.DNSSvc.DeleteByName(parent.ID, subPrefix)
			provisionDNSZone(h.DNSSvc, h.AgentClient, parent.ID, parent.Name)
		}
	}

	// Delete from DB (cascades to child domains, dns_records)
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

	// Remove NGINX config files (domain + mail subdomain)
	for _, vhostName := range []string{d.Name, "mail." + d.Name} {
		h.AgentClient.Call("file_delete", map[string]interface{}{
			"path": fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", vhostName),
		})
		h.AgentClient.Call("file_delete", map[string]interface{}{
			"path": fmt.Sprintf("/etc/nginx/sites-available/%s.conf", vhostName),
		})
	}

	// Remove SSL certificate directories (domain + mail subdomain)
	for _, sslName := range []string{d.Name, "mail." + d.Name} {
		h.AgentClient.Call("ssl_delete_cert", map[string]interface{}{
			"domain": sslName,
		})
	}

	// Reload NGINX
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Printf("ERROR: nginx reload failed after deleting %s: %v", d.Name, err)
	}

	// Remove DNS records and zone
	if d.ParentID == nil || d.SeparateDNS {
		if err := h.DNSSvc.DeleteByDomain(id); err != nil {
			log.Printf("WARNING: failed to delete DNS records for %s: %v", d.Name, err)
		}
	}
	// Always try to remove the zone file/config — handles stale zones from
	// previous separate_dns toggles or prior creations
	h.AgentClient.Call("dns_remove_zone", map[string]interface{}{"domain": d.Name})
	h.AgentClient.Call("dns_reload", nil)

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

// checkDomainAccess verifies the current user has access to a domain.
// Super admins can access all domains; others can only access their own.
func checkDomainAccess(c *fiber.Ctx, d *domain.Domain) bool {
	if middleware.IsSuperAdmin(c) {
		return true
	}
	adminID, _ := c.Locals("admin_id").(int64)
	return d.AdminID != nil && *d.AdminID == adminID
}

// containsSuffix checks if name ends with ".suffix"
func containsSuffix(name, suffix string) bool {
	return len(name) > len(suffix) && name[len(name)-len(suffix)-1:] == "."+suffix
}

// extractSubPrefix extracts the subdomain prefix from an FQDN given the parent domain name.
// e.g. extractSubPrefix("blog.example.com", "example.com") => "blog"
func extractSubPrefix(fqdn, parentName string) string {
	if len(fqdn) > len(parentName)+1 {
		return fqdn[:len(fqdn)-len(parentName)-1]
	}
	return fqdn
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

// getServerIPv6 returns the server's primary public IPv6 address.
// Returns empty string if no public IPv6 is available.
func getServerIPv6() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		// Skip IPv4
		if ipNet.IP.To4() != nil {
			continue
		}
		ip := ipNet.IP.String()
		// Skip link-local (fe80::)
		if ipNet.IP.IsLinkLocalUnicast() || ipNet.IP.IsLinkLocalMulticast() {
			continue
		}
		// Prefer non-private (global unicast)
		if !ipNet.IP.IsPrivate() {
			return ip
		}
	}
	return ""
}

// checkSystemHealth verifies that critical services are working after domain
// provisioning and returns any warnings for the user.
func (h *DomainHandler) checkSystemHealth() []string {
	var warnings []string

	// Check NGINX
	if _, err := h.AgentClient.Call("service_status", map[string]interface{}{"service": "nginx"}); err != nil {
		warnings = append(warnings, "NGINX is not responding")
	}

	// Check BIND/DNS
	dnsOk := false
	if resp, err := h.AgentClient.Call("service_status", map[string]interface{}{"service": "named"}); err == nil {
		if m, ok := resp.Result.(map[string]interface{}); ok {
			if active, ok := m["active"].(bool); ok && active {
				dnsOk = true
			}
		}
	}
	if !dnsOk {
		if resp, err := h.AgentClient.Call("service_status", map[string]interface{}{"service": "bind9"}); err == nil {
			if m, ok := resp.Result.(map[string]interface{}); ok {
				if active, ok := m["active"].(bool); ok && active {
					dnsOk = true
				}
			}
		}
	}
	if !dnsOk {
		warnings = append(warnings, "DNS server (BIND9) is not running — DNS records will not resolve")
	}

	// Check PHP-FPM
	if _, err := h.AgentClient.Call("service_status", map[string]interface{}{"service": "php8.3-fpm"}); err != nil {
		warnings = append(warnings, "PHP-FPM is not responding")
	}

	// Check MariaDB
	if _, err := h.AgentClient.Call("service_status", map[string]interface{}{"service": "mariadb"}); err != nil {
		warnings = append(warnings, "MariaDB is not responding")
	}

	return warnings
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

// ToggleModSecurity enables or disables ModSecurity WAF for a domain.
func (h *DomainHandler) ToggleModSecurity(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid domain ID",
			},
		})
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
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

	if !checkDomainAccess(c, d) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	d, err = h.DomainSvc.ToggleModSecurity(domainID, req.Enabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
	}

	// Re-render NGINX vhost
	vhostData := tmpl.NginxVhostData{
		Domain:             d.Name,
		DocumentRoot:       d.DocumentRoot,
		PHPVersion:         d.PHPVersion,
		ModSecurityEnabled: req.Enabled,
	}

	// Preserve SSL settings if present
	var certPath, keyPath string
	var chainPath *string
	var forceHTTPS bool
	err = h.DB.QueryRow(
		"SELECT cert_path, key_path, chain_path, force_https FROM ssl_certificates WHERE domain_id = ?", domainID,
	).Scan(&certPath, &keyPath, &chainPath, &forceHTTPS)
	if err == nil {
		vhostData.SSLEnabled = true
		vhostData.SSLCertPath = certPath
		vhostData.SSLKeyPath = keyPath
		if chainPath != nil {
			vhostData.SSLChainPath = *chainPath
		}
		vhostData.ForceHTTPS = forceHTTPS
		vhostData.HTTP2 = true
		vhostData.HSTS = true
		vhostData.HSTSMaxAge = 31536000
	}

	vhostContent, err := tmpl.RenderNginxVhost(vhostData)
	if err != nil {
		log.Printf("WARNING: failed to render vhost for %s: %v", d.Name, err)
		return c.JSON(fiber.Map{"status": "ok", "modsecurity_enabled": req.Enabled})
	}

	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", d.Name)
	h.AgentClient.Call("file_write", map[string]any{"path": configPath, "content": vhostContent, "mode": "0644"})
	h.AgentClient.Call("file_write", map[string]any{"path": enabledPath, "content": vhostContent, "mode": "0644"})

	if _, err := h.AgentClient.Call("nginx_test", nil); err != nil {
		log.Printf("WARNING: nginx config test failed after toggling ModSecurity for %s: %v", d.Name, err)
	} else {
		if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
			log.Printf("ERROR: nginx reload failed after toggling ModSecurity for %s: %v", d.Name, err)
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "toggle_modsecurity", "domain", d.ID, fmt.Sprintf("modsecurity=%v", req.Enabled), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "modsecurity_enabled": req.Enabled})
}
