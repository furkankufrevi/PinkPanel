package handlers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type FileHandler struct {
	DB          *sql.DB
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// basePath returns the allowed base directory for a domain (/var/www/<domain>/).
func (h *FileHandler) basePath(dom *domain.Domain) string {
	// DocumentRoot is typically /var/www/<domain>/public, parent is the site root
	return filepath.Dir(dom.DocumentRoot)
}

// validateFilePath ensures the path is within the domain's allowed directory.
func (h *FileHandler) validateFilePath(dom *domain.Domain, path string) (string, error) {
	base := h.basePath(dom)
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, base) {
		return "", fmt.Errorf("path is outside domain directory")
	}
	return cleaned, nil
}

// getDomain is a helper to parse domain ID and fetch domain.
func (h *FileHandler) getDomain(c *fiber.Ctx) (*domain.Domain, error) {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid domain ID")
	}
	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return nil, fmt.Errorf("domain not found")
	}
	return dom, nil
}

// List returns directory contents.
func (h *FileHandler) List(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	path := c.Query("path", h.basePath(dom))
	cleanPath, err := h.validateFilePath(dom, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	resp, err := h.AgentClient.Call("file_list", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"data": resp.Result, "path": cleanPath, "base": h.basePath(dom)})
}

// Read returns file contents.
func (h *FileHandler) Read(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}
	cleanPath, err := h.validateFilePath(dom, path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	resp, err := h.AgentClient.Call("file_read", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(resp.Result)
}

// Save creates or updates a file.
func (h *FileHandler) Save(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}

	cleanPath, err := h.validateFilePath(dom, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	mode := req.Mode
	if mode == "" {
		mode = "0644"
	}
	if _, err := h.AgentClient.Call("file_write", map[string]any{
		"path": cleanPath, "content": req.Content, "mode": mode,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_save", "domain", dom.ID, cleanPath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Delete removes a file or directory.
func (h *FileHandler) Delete(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}

	cleanPath, err := h.validateFilePath(dom, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	// Prevent deleting the base directory itself
	if cleanPath == h.basePath(dom) {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": "cannot delete site root directory"}})
	}

	if _, err := h.AgentClient.Call("file_delete", map[string]any{
		"path": cleanPath, "recursive": req.Recursive,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_delete", "domain", dom.ID, cleanPath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Rename renames or moves a file/directory.
func (h *FileHandler) Rename(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	oldClean, err := h.validateFilePath(dom, req.OldPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}
	newClean, err := h.validateFilePath(dom, req.NewPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_rename", map[string]any{
		"old_path": oldClean, "new_path": newClean,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_rename", "domain", dom.ID, fmt.Sprintf("%s -> %s", oldClean, newClean), c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Copy copies a file or directory.
func (h *FileHandler) Copy(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Source string `json:"source"`
		Dest   string `json:"dest"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	srcClean, err := h.validateFilePath(dom, req.Source)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}
	destClean, err := h.validateFilePath(dom, req.Dest)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_copy", map[string]any{
		"source": srcClean, "dest": destClean,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// CreateDirectory creates a new directory.
func (h *FileHandler) CreateDirectory(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}

	cleanPath, err := h.validateFilePath(dom, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("dir_create", map[string]any{
		"path": cleanPath, "owner": "www-data", "group": "www-data",
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "dir_create", "domain", dom.ID, cleanPath, c.IP())

	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

// Extract extracts an archive.
func (h *FileHandler) Extract(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Archive string `json:"archive"`
		Dest    string `json:"dest"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	archiveClean, err := h.validateFilePath(dom, req.Archive)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}
	destClean, err := h.validateFilePath(dom, req.Dest)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_extract", map[string]any{
		"archive": archiveClean, "dest": destClean,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_extract", "domain", dom.ID, archiveClean, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// SetPermissions changes file/directory permissions.
func (h *FileHandler) SetPermissions(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Path      string `json:"path"`
		Mode      string `json:"mode"`
		Recursive bool   `json:"recursive"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	cleanPath, err := h.validateFilePath(dom, req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("set_permissions", map[string]any{
		"path": cleanPath, "mode": req.Mode, "recursive": req.Recursive,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
