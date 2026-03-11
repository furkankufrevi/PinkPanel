package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/ftp"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type FTPHandler struct {
	DB          *sql.DB
	FTPSvc      *ftp.Service
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// List returns FTP accounts, optionally filtered by domain.
func (h *FTPHandler) List(c *fiber.Ctx) error {
	var domainID *int64
	if did := c.Query("domain_id"); did != "" {
		id, err := strconv.ParseInt(did, 10, 64)
		if err == nil {
			domainID = &id
		}
	}

	accounts, err := h.FTPSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if accounts == nil {
		accounts = []ftp.Account{}
	}
	return c.JSON(fiber.Map{"data": accounts})
}

// Get returns a single FTP account.
func (h *FTPHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid FTP account ID"}})
	}
	a, err := h.FTPSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(a)
}

// Create creates an FTP account.
func (h *FTPHandler) Create(c *fiber.Ctx) error {
	var req struct {
		DomainID int64  `json:"domain_id"`
		Username string `json:"username"`
		Password string `json:"password"`
		HomeDir  string `json:"home_dir"`
		QuotaMB  int64  `json:"quota_mb"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "username and password are required"}})
	}
	if req.DomainID == 0 {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "domain_id is required"}})
	}

	// Resolve home directory from domain if not specified
	if req.HomeDir == "" {
		dom, err := h.DomainSvc.GetByID(req.DomainID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
		}
		req.HomeDir = dom.DocumentRoot
	}

	// Create system FTP user via agent
	if _, err := h.AgentClient.Call("ftp_create_user", map[string]any{
		"username": req.Username,
		"password": req.Password,
		"home_dir": req.HomeDir,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to create FTP user: " + err.Error()}})
	}

	// Store in panel database
	a, err := h.FTPSvc.Create(req.DomainID, req.Username, req.HomeDir, req.QuotaMB)
	if err != nil {
		// Rollback: delete system user
		if _, rollbackErr := h.AgentClient.Call("ftp_delete_user", map[string]any{"username": req.Username}); rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("failed to rollback FTP user creation")
		}
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	// Reload FTP server
	if _, err := h.AgentClient.Call("ftp_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload FTP server")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_ftp_account", "ftp", a.ID, req.Username, c.IP())

	return c.Status(201).JSON(a)
}

// Delete removes an FTP account.
func (h *FTPHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid FTP account ID"}})
	}

	a, err := h.FTPSvc.Delete(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Delete system FTP user via agent
	if _, err := h.AgentClient.Call("ftp_delete_user", map[string]any{"username": a.Username}); err != nil {
		log.Error().Err(err).Str("username", a.Username).Msg("failed to delete FTP user via agent")
	}

	// Reload FTP server
	if _, err := h.AgentClient.Call("ftp_reload", nil); err != nil {
		log.Error().Err(err).Msg("failed to reload FTP server")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_ftp_account", "ftp", id, a.Username, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// UpdateQuota updates the quota for an FTP account.
func (h *FTPHandler) UpdateQuota(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid FTP account ID"}})
	}

	var req struct {
		QuotaMB int64 `json:"quota_mb"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	if err := h.FTPSvc.UpdateQuota(id, req.QuotaMB); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
