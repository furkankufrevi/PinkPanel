package handlers

import (
	"database/sql"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type SettingsHandler struct {
	DB          *sql.DB
	AgentClient *agent.Client
	Version     string
}

// ActivityLog returns recent activity log entries.
func (h *SettingsHandler) ActivityLog(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	entries, err := db.RecentActivity(h.DB, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if entries == nil {
		entries = []db.ActivityEntry{}
	}
	return c.JSON(fiber.Map{"data": entries})
}

// ServerInfo returns system information from the agent.
func (h *SettingsHandler) ServerInfo(c *fiber.Ctx) error {
	resp, err := h.AgentClient.Call("system_info", nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to get system info: " + err.Error()}})
	}

	return c.JSON(fiber.Map{
		"panel_version": h.Version,
		"system":        resp.Result,
	})
}
