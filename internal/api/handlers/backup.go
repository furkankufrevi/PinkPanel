package handlers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/backup"
	dbpkg "github.com/pinkpanel/pinkpanel/internal/core/database"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

const backupBaseDir = "/var/backups/pinkpanel"

type BackupHandler struct {
	DB          *sql.DB
	BackupSvc   *backup.Service
	DomainSvc   *domain.Service
	DBSvc       *dbpkg.Service
	AgentClient *agent.Client
}

// List returns all backups, optionally filtered by domain.
func (h *BackupHandler) List(c *fiber.Ctx) error {
	var domainID *int64
	if did := c.Query("domain_id"); did != "" {
		id, err := strconv.ParseInt(did, 10, 64)
		if err == nil {
			domainID = &id
		}
	}

	backups, err := h.BackupSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if backups == nil {
		backups = []backup.Backup{}
	}
	return c.JSON(fiber.Map{"data": backups})
}

// Get returns a single backup.
func (h *BackupHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid backup ID"}})
	}
	b, err := h.BackupSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(b)
}

// Create initiates a new backup.
func (h *BackupHandler) Create(c *fiber.Ctx) error {
	var req struct {
		Type     string `json:"type"`
		DomainID *int64 `json:"domain_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Type == "" {
		req.Type = "full"
	}

	// Build backup parameters
	var sourcePaths []string
	var databases []string
	timestamp := time.Now().UTC().Format("20060102-150405")
	var fileName string

	if req.Type == "domain" {
		if req.DomainID == nil {
			return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "domain_id is required for domain backups"}})
		}
		dom, err := h.DomainSvc.GetByID(*req.DomainID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
		}
		sourcePaths = append(sourcePaths, filepath.Dir(dom.DocumentRoot))
		// Include domain databases
		dbs, _ := h.DBSvc.List(req.DomainID)
		for _, d := range dbs {
			databases = append(databases, d.Name)
		}
		fileName = fmt.Sprintf("domain-%s-%s.tar.gz", dom.Name, timestamp)
	} else {
		// Full backup: all domains
		domains, _, _ := h.DomainSvc.List("", "", 1, 10000)
		for _, d := range domains {
			sourcePaths = append(sourcePaths, filepath.Dir(d.DocumentRoot))
		}
		// All databases
		allDBs, _ := h.DBSvc.List(nil)
		for _, d := range allDBs {
			databases = append(databases, d.Name)
		}
		fileName = fmt.Sprintf("full-%s.tar.gz", timestamp)
	}

	filePath := filepath.Join(backupBaseDir, fileName)

	// Create backup record
	b, err := h.BackupSvc.Create(req.DomainID, req.Type, filePath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Run backup via agent (async-ish: we start it and update status)
	go func() {
		if err := h.BackupSvc.UpdateStatus(b.ID, "running", 0); err != nil {
			log.Error().Err(err).Msg("failed to update backup status to running")
		}

		resp, err := h.AgentClient.Call("backup_create", map[string]any{
			"source_paths": sourcePaths,
			"databases":    databases,
			"output":       filePath,
		})
		if err != nil {
			log.Error().Err(err).Int64("backup_id", b.ID).Msg("backup failed")
			h.BackupSvc.UpdateStatus(b.ID, "failed", 0)
			return
		}

		// Extract size from response if available
		var sizeBytes int64
		if result, ok := resp.Result.(map[string]any); ok {
			if sz, ok := result["size_bytes"].(float64); ok {
				sizeBytes = int64(sz)
			}
		}
		h.BackupSvc.UpdateStatus(b.ID, "completed", sizeBytes)
	}()

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_backup", "backup", b.ID, fileName, c.IP())

	return c.Status(201).JSON(b)
}

// Delete removes a backup.
func (h *BackupHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid backup ID"}})
	}

	b, err := h.BackupSvc.Delete(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Delete backup file via agent
	if _, err := h.AgentClient.Call("backup_delete", map[string]any{"path": b.FilePath}); err != nil {
		log.Error().Err(err).Str("path", b.FilePath).Msg("failed to delete backup file via agent")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_backup", "backup", id, b.FilePath, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// Download serves a backup file for download.
func (h *BackupHandler) Download(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid backup ID"}})
	}

	b, err := h.BackupSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	if b.Status != "completed" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "can only download completed backups"}})
	}

	// Copy backup to a temp file the server process can read
	tmpPath := fmt.Sprintf("/tmp/pinkpanel-backup-dl-%d-%d.tar.gz", id, time.Now().UnixNano())
	if _, err := h.AgentClient.Call("file_copy", map[string]any{
		"source": b.FilePath,
		"dest":   tmpPath,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to prepare backup for download: " + err.Error()}})
	}
	// Make temp file readable by server process
	if _, err := h.AgentClient.Call("set_permissions", map[string]any{
		"path": tmpPath,
		"mode": "644",
	}); err != nil {
		log.Error().Err(err).Msg("failed to set permissions on temp backup file")
	}
	// Clean up temp file after response
	defer func() {
		h.AgentClient.Call("file_delete", map[string]any{"path": tmpPath, "recursive": false})
	}()

	fileName := filepath.Base(b.FilePath)
	c.Set("Content-Type", "application/gzip")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))

	return c.SendFile(tmpPath)
}

// Restore initiates a backup restore.
func (h *BackupHandler) Restore(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid backup ID"}})
	}

	b, err := h.BackupSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	if b.Status != "completed" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "can only restore completed backups"}})
	}

	// Determine restore destination
	dest := "/var/www"
	if b.DomainID != nil {
		dom, err := h.DomainSvc.GetByID(*b.DomainID)
		if err == nil {
			dest = filepath.Dir(dom.DocumentRoot)
		}
	}

	if _, err := h.AgentClient.Call("backup_restore", map[string]any{
		"archive": b.FilePath,
		"dest":    dest,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "restore failed: " + err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "restore_backup", "backup", id, b.FilePath, c.IP())

	return c.JSON(fiber.Map{"status": "ok", "message": "Backup restored successfully"})
}
