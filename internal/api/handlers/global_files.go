package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

// GlobalFileHandler handles file operations across all websites from /var/www.
type GlobalFileHandler struct {
	DB          *sql.DB
	AgentClient *agent.Client
}

const globalBasePath = "/var/www"

// validateGlobalPath ensures the path is within /var/www.
func validateGlobalPath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if !strings.HasPrefix(cleaned, globalBasePath) {
		return "", fmt.Errorf("path is outside /var/www")
	}
	return cleaned, nil
}

func (h *GlobalFileHandler) List(c *fiber.Ctx) error {
	path := c.Query("path", globalBasePath)
	cleanPath, err := validateGlobalPath(path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	resp, err := h.AgentClient.Call("file_list", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"data": resp.Result, "path": cleanPath, "base": globalBasePath})
}

func (h *GlobalFileHandler) Read(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}
	cleanPath, err := validateGlobalPath(path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	resp, err := h.AgentClient.Call("file_read", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	return c.JSON(resp.Result)
}

func (h *GlobalFileHandler) Save(c *fiber.Ctx) error {
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

	cleanPath, err := validateGlobalPath(req.Path)
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
	db.LogActivity(h.DB, adminID, "file_save", "global", 0, cleanPath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) Delete(c *fiber.Ctx) error {
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

	cleanPath, err := validateGlobalPath(req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if cleanPath == globalBasePath {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": "cannot delete /var/www"}})
	}

	if _, err := h.AgentClient.Call("file_delete", map[string]any{
		"path": cleanPath, "recursive": req.Recursive,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_delete", "global", 0, cleanPath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) Rename(c *fiber.Ctx) error {
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	oldClean, err := validateGlobalPath(req.OldPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}
	newClean, err := validateGlobalPath(req.NewPath)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_rename", map[string]any{
		"old_path": oldClean, "new_path": newClean,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_rename", "global", 0, fmt.Sprintf("%s -> %s", oldClean, newClean), c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) CreateDirectory(c *fiber.Ctx) error {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}

	cleanPath, err := validateGlobalPath(req.Path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("dir_create", map[string]any{
		"path": cleanPath, "owner": "www-data", "group": "www-data",
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "dir_create", "global", 0, cleanPath, c.IP())

	return c.Status(201).JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) Extract(c *fiber.Ctx) error {
	var req struct {
		Archive string `json:"archive"`
		Dest    string `json:"dest"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	archiveClean, err := validateGlobalPath(req.Archive)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}
	destClean, err := validateGlobalPath(req.Dest)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_extract", map[string]any{
		"archive": archiveClean, "dest": destClean,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_extract", "global", 0, archiveClean, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) Upload(c *fiber.Ctx) error {
	destPath := c.FormValue("path", globalBasePath)
	cleanDest, err := validateGlobalPath(destPath)
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
		if _, err := validateGlobalPath(finalPath); err != nil {
			os.Remove(tmpPath)
			return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
		}

		if _, err := h.AgentClient.Call("file_rename", map[string]any{
			"old_path": tmpPath, "new_path": finalPath,
		}); err != nil {
			os.Remove(tmpPath)
			return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
		}

		h.AgentClient.Call("set_ownership", map[string]any{
			"path": finalPath, "owner": "www-data", "group": "www-data",
		})

		uploaded = append(uploaded, file.Filename)
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_upload", "global", 0, strings.Join(uploaded, ", "), c.IP())

	return c.JSON(fiber.Map{"status": "ok", "uploaded": uploaded})
}

func (h *GlobalFileHandler) Download(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "path is required"}})
	}
	cleanPath, err := validateGlobalPath(path)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	// Check if directory
	resp, err := h.AgentClient.Call("file_list", map[string]any{"path": cleanPath})
	if err == nil && resp.Result != nil {
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

	readResp, err := h.AgentClient.Call("file_read", map[string]any{"path": cleanPath})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	content := ""
	if m, ok := readResp.Result.(map[string]interface{}); ok {
		if c2, ok := m["content"].(string); ok {
			content = c2
		}
	}

	name := filepath.Base(cleanPath)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	c.Set("Content-Type", "application/octet-stream")

	return c.SendString(content)
}

func (h *GlobalFileHandler) Compress(c *fiber.Ctx) error {
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

	for _, src := range req.Sources {
		if _, err := validateGlobalPath(src); err != nil {
			return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
		}
	}
	cleanOutput, err := validateGlobalPath(req.Output)
	if err != nil {
		return c.Status(403).JSON(fiber.Map{"error": fiber.Map{"code": "forbidden", "message": err.Error()}})
	}

	if _, err := h.AgentClient.Call("file_compress", map[string]any{
		"sources": req.Sources, "output": cleanOutput, "format": req.Format,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "file_compress", "global", 0, cleanOutput, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *GlobalFileHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "q parameter is required"}})
	}

	searchPath := c.Query("path", globalBasePath)
	cleanPath, err := validateGlobalPath(searchPath)
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
