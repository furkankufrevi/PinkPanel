package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// Upload handles file uploads via multipart/form-data.
func (h *FileHandler) Upload(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	destPath := c.FormValue("path", h.basePath(dom))
	cleanDest, err := h.validateFilePath(dom, destPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid multipart form"}})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "no files provided"}})
	}

	uploaded := make([]string, 0, len(files))
	for _, file := range files {
		tmpPath := fmt.Sprintf("/tmp/pinkpanel-upload-%d-%s", time.Now().UnixNano(), file.Filename)
		if err := c.SaveFile(file, tmpPath); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "upload_error", "message": fmt.Sprintf("failed to save %s: %v", file.Filename, err)}})
		}

		finalPath := filepath.Join(cleanDest, file.Filename)
		if _, err := h.validateFilePath(dom, finalPath); err != nil {
			os.Remove(tmpPath)
			return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
		}

		if _, err := h.AgentClient.Call("file_rename", map[string]any{
			"old_path": tmpPath, "new_path": finalPath,
		}); err != nil {
			os.Remove(tmpPath)
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
		}

		if _, err := h.AgentClient.Call("set_ownership", map[string]any{
			"path": finalPath, "owner": "www-data", "group": "www-data",
		}); err != nil {
			// Non-fatal, file is already uploaded
		}

		uploaded = append(uploaded, file.Filename)
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_upload", "domain", dom.ID, strings.Join(uploaded, ", "), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "uploaded": uploaded})
}

// Download serves a file or directory (as zip) for download.
func (h *FileHandler) Download(c *fiber.Ctx) error {
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

	// Check if it's a directory by trying to list it
	resp, err := h.AgentClient.Call("file_list", map[string]any{"path": cleanPath})
	if err == nil && resp.Result != nil {
		// It's a directory — compress to temp zip and serve
		tmpZip := fmt.Sprintf("/tmp/pinkpanel-dl-%d.zip", time.Now().UnixNano())
		if _, err := h.AgentClient.Call("file_compress", map[string]any{
			"sources": []string{cleanPath}, "output": tmpZip, "format": "zip",
		}); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
		}
		defer func() {
			h.AgentClient.Call("file_delete", map[string]any{"path": tmpZip, "recursive": false})
		}()

		name := filepath.Base(cleanPath) + ".zip"
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		c.Set("Content-Type", "application/zip")
		return c.SendFile(tmpZip)
	}

	// It's a file — read and send
	readResp, err := h.AgentClient.Call("file_read", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	// Extract content from response
	content := ""
	if m, ok := readResp.Result.(map[string]interface{}); ok {
		if c2, ok := m["content"].(string); ok {
			content = c2
		}
	}

	name := filepath.Base(cleanPath)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	// Detect content type from extension
	ext := strings.ToLower(filepath.Ext(name))
	contentType := "application/octet-stream"
	switch ext {
	case ".html", ".htm":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".svg":
		contentType = "image/svg+xml"
	case ".pdf":
		contentType = "application/pdf"
	case ".zip":
		contentType = "application/zip"
	case ".txt", ".log", ".md":
		contentType = "text/plain"
	case ".xml":
		contentType = "application/xml"
	case ".php":
		contentType = "text/x-php"
	}
	c.Set("Content-Type", contentType)

	return c.SendString(content)
}

// Compress creates an archive from selected files/directories.
func (h *FileHandler) Compress(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	var req struct {
		Sources []string `json:"sources"`
		Output  string   `json:"output"`
		Format  string   `json:"format"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if len(req.Sources) == 0 || req.Output == "" || req.Format == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "sources, output, and format are required"}})
	}

	// Validate all paths
	for _, src := range req.Sources {
		if _, err := h.validateFilePath(dom, src); err != nil {
			return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
		}
	}
	cleanOutput, err := h.validateFilePath(dom, req.Output)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_compress", map[string]any{
		"sources": req.Sources, "output": cleanOutput, "format": req.Format,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_compress", "domain", dom.ID, cleanOutput, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Search searches file contents within a domain directory.
func (h *FileHandler) Search(c *fiber.Ctx) error {
	dom, err := h.getDomain(c)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "q parameter is required"}})
	}

	searchPath := c.Query("path", h.basePath(dom))
	cleanPath, err := h.validateFilePath(dom, searchPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	resp, err := h.AgentClient.Call("file_search", map[string]any{
		"path": cleanPath, "query": query, "max_results": 100,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"data": resp.Result})
}
