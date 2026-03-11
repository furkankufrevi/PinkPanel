package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/db"
	tmpl "github.com/pinkpanel/pinkpanel/internal/template"
)

// DomainHandler handles domain CRUD and lifecycle operations.
type DomainHandler struct {
	DB          *sql.DB
	DomainSvc   *domain.Service
	AgentClient *agent.Client
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

	// Render NGINX vhost configuration
	vhostContent, err := tmpl.RenderNginxVhost(tmpl.NginxVhostData{
		Domain:       d.Name,
		DocumentRoot: d.DocumentRoot,
		PHPVersion:   d.PHPVersion,
	})
	if err != nil {
		log.Printf("WARNING: failed to render vhost for %s: %v", d.Name, err)
		return c.Status(fiber.StatusCreated).JSON(d)
	}

	// Write vhost to sites-available
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", d.Name)
	_, err = h.AgentClient.Call("file_write", map[string]interface{}{
		"path":    configPath,
		"content": vhostContent,
		"mode":    "0644",
	})
	if err != nil {
		log.Printf("WARNING: failed to write vhost config for %s: %v", d.Name, err)
		return c.Status(fiber.StatusCreated).JSON(d)
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

	// Test NGINX configuration
	_, err = h.AgentClient.Call("nginx_test", nil)
	if err != nil {
		log.Printf("WARNING: nginx config test failed after creating %s: %v", d.Name, err)
	}

	// Reload NGINX
	_, err = h.AgentClient.Call("nginx_reload", nil)
	if err != nil {
		log.Printf("ERROR: nginx reload failed after creating %s: %v", d.Name, err)
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

	// Delete from DB
	if err := h.DomainSvc.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to delete domain",
			},
		})
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
