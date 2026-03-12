package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/core/backup"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type BackupScheduleHandler struct {
	DB          *sql.DB
	ScheduleSvc *backup.ScheduleService
}

func (h *BackupScheduleHandler) List(c *fiber.Ctx) error {
	schedules, err := h.ScheduleSvc.List()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if schedules == nil {
		schedules = []backup.Schedule{}
	}
	return c.JSON(fiber.Map{"data": schedules})
}

func (h *BackupScheduleHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid schedule ID"}})
	}
	sc, err := h.ScheduleSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	return c.JSON(sc)
}

func (h *BackupScheduleHandler) Create(c *fiber.Ctx) error {
	var req struct {
		DomainID       *int64 `json:"domain_id"`
		Frequency      string `json:"frequency"`
		Time           string `json:"time"`
		RetentionCount int    `json:"retention_count"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Frequency == "" {
		req.Frequency = "daily"
	}
	if req.RetentionCount < 1 {
		req.RetentionCount = 5
	}

	sc, err := h.ScheduleSvc.Create(req.DomainID, req.Frequency, req.Time, req.RetentionCount)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_backup_schedule", "backup_schedule", sc.ID, req.Frequency, c.IP())

	return c.Status(201).JSON(sc)
}

func (h *BackupScheduleHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid schedule ID"}})
	}

	var req struct {
		Frequency      string `json:"frequency"`
		Time           string `json:"time"`
		RetentionCount int    `json:"retention_count"`
		Enabled        bool   `json:"enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}

	sc, err := h.ScheduleSvc.Update(id, req.Frequency, req.Time, req.RetentionCount, req.Enabled)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "update_backup_schedule", "backup_schedule", id, req.Frequency, c.IP())

	return c.JSON(sc)
}

func (h *BackupScheduleHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid schedule ID"}})
	}

	if err := h.ScheduleSvc.Delete(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_backup_schedule", "backup_schedule", id, "", c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}
