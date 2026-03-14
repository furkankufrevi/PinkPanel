package handlers

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/redirect"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type RedirectHandler struct {
	DB          *sql.DB
	RedirectSvc *redirect.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// List returns redirects for a domain.
func (h *RedirectHandler) List(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	redirects, err := h.RedirectSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if redirects == nil {
		redirects = []redirect.Redirect{}
	}
	return c.JSON(fiber.Map{"data": redirects})
}

// Get returns a single redirect.
func (h *RedirectHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid redirect ID"}})
	}
	r, err := h.RedirectSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(r)
}

// Create creates a redirect and syncs nginx.
func (h *RedirectHandler) Create(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		SourcePath   string `json:"source_path"`
		TargetURL    string `json:"target_url"`
		RedirectType int    `json:"redirect_type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.RedirectType == 0 {
		req.RedirectType = 301
	}

	r, err := h.RedirectSvc.Create(domainID, req.SourcePath, req.TargetURL, req.RedirectType)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	h.syncNginx(domainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_redirect", "redirect", r.ID, r.SourcePath, c.IP())

	return c.Status(201).JSON(r)
}

// Update updates a redirect and syncs nginx.
func (h *RedirectHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid redirect ID"}})
	}

	var req struct {
		SourcePath   *string `json:"source_path"`
		TargetURL    *string `json:"target_url"`
		RedirectType *int    `json:"redirect_type"`
		Enabled      *bool   `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	r, err := h.RedirectSvc.Update(id, req.SourcePath, req.TargetURL, req.RedirectType, req.Enabled)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	h.syncNginx(r.DomainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_redirect", "redirect", r.ID, r.SourcePath, c.IP())

	return c.JSON(r)
}

// Delete removes a redirect and syncs nginx.
func (h *RedirectHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid redirect ID"}})
	}

	r, err := h.RedirectSvc.Delete(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	h.syncNginx(r.DomainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_redirect", "redirect", id, r.SourcePath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// syncNginx generates the nginx redirect snippet and reloads.
func (h *RedirectHandler) syncNginx(domainID int64) {
	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		log.Error().Err(err).Int64("domain_id", domainID).Msg("redirect: failed to get domain")
		return
	}

	redirects, err := h.RedirectSvc.ListEnabled(domainID)
	if err != nil {
		log.Error().Err(err).Msg("redirect: failed to list enabled redirects")
		return
	}

	// Generate nginx snippet
	var lines []string
	lines = append(lines, "# PinkPanel managed redirects - do not edit manually")
	for _, r := range redirects {
		// Escape special chars in source path for nginx location matching
		escapedPath := strings.ReplaceAll(r.SourcePath, "'", "\\'")
		lines = append(lines, fmt.Sprintf("location = %s { return %d %s; }", escapedPath, r.RedirectType, r.TargetURL))
	}
	content := strings.Join(lines, "\n") + "\n"

	snippetPath := fmt.Sprintf("/etc/nginx/snippets/redirects-%s.conf", dom.Name)

	// Write snippet via agent
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path":    snippetPath,
		"content": content,
		"mode":    "0644",
	}); err != nil {
		log.Error().Err(err).Str("path", snippetPath).Msg("redirect: failed to write nginx snippet")
		return
	}

	// Ensure snippet is included in ALL server blocks of the domain's vhost.
	// SSL configs have multiple server blocks (HTTP redirect + HTTPS content);
	// the include must be in every block so redirects work on both ports.
	vhostPath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", dom.Name)
	resp, err := h.AgentClient.Call("file_read", map[string]any{"path": vhostPath})
	if err == nil {
		if result, ok := resp.Result.(map[string]any); ok {
			if vhostContent, ok := result["content"].(string); ok {
				includeLine := fmt.Sprintf("    include %s;", snippetPath)
				// Strip any existing include lines to avoid duplicates
				vhostContent = strings.ReplaceAll(vhostContent, includeLine+"\n", "")
				// Inject include after every "server {" so all blocks have it
				vhostContent = strings.ReplaceAll(
					vhostContent,
					"server {",
					"server {\n"+includeLine,
				)
				if _, err := h.AgentClient.Call("file_write", map[string]any{
					"path":    vhostPath,
					"content": vhostContent,
					"mode":    "0644",
				}); err != nil {
					log.Error().Err(err).Msg("redirect: failed to update vhost")
				}
			}
		}
	}

	// Reload nginx
	if _, err := h.AgentClient.Call("nginx_reload", nil); err != nil {
		log.Error().Err(err).Msg("redirect: failed to reload nginx")
	}
}
