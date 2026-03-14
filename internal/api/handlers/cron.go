package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/core/cron"
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
	"github.com/pinkpanel/pinkpanel/internal/core/user"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type CronHandler struct {
	DB          *sql.DB
	CronSvc     *cron.Service
	DomainSvc   *domain.Service
	UserSvc     *user.Service
	AgentClient *agent.Client
}

// List returns cron jobs for a domain.
func (h *CronHandler) List(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	jobs, err := h.CronSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if jobs == nil {
		jobs = []cron.CronJob{}
	}
	return c.JSON(fiber.Map{"data": jobs})
}

// Get returns a single cron job.
func (h *CronHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid cron job ID"}})
	}
	j, err := h.CronSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(j)
}

// Create creates a cron job and syncs the crontab.
func (h *CronHandler) Create(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	var req struct {
		Schedule    string `json:"schedule"`
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Schedule == "" || req.Command == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "schedule and command are required"}})
	}

	j, err := h.CronSvc.Create(domainID, req.Schedule, req.Command, req.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	h.syncCrontab(domainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_cron_job", "cron", j.ID, j.Description, c.IP())

	return c.Status(201).JSON(j)
}

// Update updates a cron job and syncs the crontab.
func (h *CronHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid cron job ID"}})
	}

	var req struct {
		Schedule    *string `json:"schedule"`
		Command     *string `json:"command"`
		Description *string `json:"description"`
		Enabled     *bool   `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	j, err := h.CronSvc.Update(id, req.Schedule, req.Command, req.Description, req.Enabled)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	h.syncCrontab(j.DomainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_cron_job", "cron", j.ID, j.Description, c.IP())

	return c.JSON(j)
}

// Delete removes a cron job and syncs the crontab.
func (h *CronHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid cron job ID"}})
	}

	j, err := h.CronSvc.Delete(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	h.syncCrontab(j.DomainID)

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_cron_job", "cron", id, j.Description, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// RunNow executes a cron job immediately and records the result.
func (h *CronHandler) RunNow(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid cron job ID"}})
	}

	j, err := h.CronSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	sysUser := h.resolveSystemUser(j.DomainID)

	resp, err := h.AgentClient.Call("cron_execute", map[string]any{
		"user":    sysUser,
		"command": j.Command,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to execute cron job: " + err.Error()}})
	}

	result := resp.Result.(map[string]any)
	exitCode := int(result["exit_code"].(float64))
	output, _ := result["output"].(string)
	durationMs := int64(result["duration_ms"].(float64))

	// Record log
	if err := h.CronSvc.CreateLog(id, exitCode, output, durationMs); err != nil {
		log.Error().Err(err).Msg("failed to record cron execution log")
	}

	return c.JSON(fiber.Map{
		"exit_code":   exitCode,
		"output":      output,
		"duration_ms": durationMs,
	})
}

// GetLogs returns execution logs for a cron job.
func (h *CronHandler) GetLogs(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid cron job ID"}})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	logs, err := h.CronSvc.ListLogs(id, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if logs == nil {
		logs = []cron.CronLog{}
	}
	return c.JSON(fiber.Map{"data": logs})
}

// resolveSystemUser returns the Linux system user for a domain.
func (h *CronHandler) resolveSystemUser(domainID int64) string {
	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return "www-data"
	}
	if dom.AdminID != nil {
		sysUser, err := h.UserSvc.GetSystemUsername(*dom.AdminID)
		if err == nil && sysUser != "" {
			return sysUser
		}
	}
	return "www-data"
}

// syncCrontab rebuilds the crontab for the domain's system user.
func (h *CronHandler) syncCrontab(domainID int64) {
	sysUser := h.resolveSystemUser(domainID)

	jobs, err := h.CronSvc.ListEnabled(domainID)
	if err != nil {
		log.Error().Err(err).Int64("domain_id", domainID).Msg("failed to list enabled cron jobs for sync")
		return
	}

	jobEntries := make([]map[string]any, len(jobs))
	for i, j := range jobs {
		jobEntries[i] = map[string]any{
			"id":       j.ID,
			"schedule": j.Schedule,
			"command":  j.Command,
		}
	}

	if _, err := h.AgentClient.Call("cron_sync", map[string]any{
		"user": sysUser,
		"jobs": jobEntries,
	}); err != nil {
		log.Error().Err(err).Str("user", sysUser).Msg("failed to sync crontab")
	}
}
