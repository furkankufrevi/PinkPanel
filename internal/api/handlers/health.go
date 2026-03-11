package handlers

import (
	"database/sql"
	"net"
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthHandler struct {
	DB          *sql.DB
	AgentSocket string
	Version     string
	StartTime   time.Time
}

type healthResponse struct {
	Status string           `json:"status"`
	Components componentStatus `json:"components"`
}

type componentStatus struct {
	Database string `json:"database"`
	Agent    string `json:"agent"`
}

type detailedHealthResponse struct {
	Status     string           `json:"status"`
	Version    string           `json:"version"`
	Uptime     string           `json:"uptime"`
	Components componentStatus  `json:"components"`
}

// Health returns basic health status (public, no auth).
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	dbStatus := "ok"
	if err := h.DB.Ping(); err != nil {
		dbStatus = "error"
	}

	agentStatus := h.checkAgent()

	overall := "ok"
	if dbStatus != "ok" {
		overall = "degraded"
	}

	return c.JSON(healthResponse{
		Status: overall,
		Components: componentStatus{
			Database: dbStatus,
			Agent:    agentStatus,
		},
	})
}

// HealthDetailed returns detailed health info (for future auth protection).
func (h *HealthHandler) HealthDetailed(c *fiber.Ctx) error {
	dbStatus := "ok"
	if err := h.DB.Ping(); err != nil {
		dbStatus = "error"
	}

	agentStatus := h.checkAgent()

	overall := "ok"
	if dbStatus != "ok" {
		overall = "degraded"
	}

	uptime := time.Since(h.StartTime).Round(time.Second).String()

	return c.JSON(detailedHealthResponse{
		Status:  overall,
		Version: h.Version,
		Uptime:  uptime,
		Components: componentStatus{
			Database: dbStatus,
			Agent:    agentStatus,
		},
	})
}

func (h *HealthHandler) checkAgent() string {
	if h.AgentSocket == "" {
		return "not_configured"
	}

	conn, err := net.DialTimeout("unix", h.AgentSocket, 2*time.Second)
	if err != nil {
		return "unreachable"
	}
	conn.Close()
	return "ok"
}
