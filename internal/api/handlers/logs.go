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
	"github.com/pinkpanel/pinkpanel/internal/core/domain"
)

type LogHandler struct {
	DB          *sql.DB
	DomainSvc   *domain.Service
	AgentClient *agent.Client
}

// logSources defines available log types and their path patterns.
var logSources = map[string]struct {
	PathFn func(domainName string) string
	Label  string
}{
	"access": {
		PathFn: func(d string) string { return filepath.Join("/var/log/nginx", d+".access.log") },
		Label:  "NGINX Access Log",
	},
	"error": {
		PathFn: func(d string) string { return filepath.Join("/var/log/nginx", d+".error.log") },
		Label:  "NGINX Error Log",
	},
	"php": {
		PathFn: func(d string) string { return filepath.Join("/var/log/php-fpm", d+".log") },
		Label:  "PHP-FPM Log",
	},
}

// Sources returns available log sources for a domain.
func (h *LogHandler) Sources(c *fiber.Ctx) error {
	type source struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	}
	sources := []source{
		{Key: "access", Label: "NGINX Access Log"},
		{Key: "error", Label: "NGINX Error Log"},
		{Key: "php", Label: "PHP-FPM Log"},
	}
	return c.JSON(fiber.Map{"data": sources})
}

// DomainLogs reads logs for a specific domain.
func (h *LogHandler) DomainLogs(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, dom) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	logType := c.Query("type", "access")
	source, ok := logSources[logType]
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": fmt.Sprintf("unknown log type: %s", logType)}})
	}

	lines, _ := strconv.Atoi(c.Query("lines", "100"))
	if lines <= 0 {
		lines = 100
	}
	if lines > 5000 {
		lines = 5000
	}
	filter := c.Query("filter", "")

	logPath := source.PathFn(dom.Name)

	resp, err := h.AgentClient.Call("log_read", map[string]any{
		"path":   logPath,
		"lines":  lines,
		"filter": filter,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to read log: " + err.Error()}})
	}

	return c.JSON(fiber.Map{
		"log_type": logType,
		"path":     logPath,
		"content":  resp.Result,
	})
}

// DownloadDomainLog serves a domain log file for download.
func (h *LogHandler) DownloadDomainLog(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	dom, err := h.DomainSvc.GetByID(domainID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "domain not found"}})
	}

	if !checkDomainAccess(c, dom) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Access denied",
			},
		})
	}

	logType := c.Query("type", "access")
	source, ok := logSources[logType]
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": fmt.Sprintf("unknown log type: %s", logType)}})
	}

	logPath := source.PathFn(dom.Name)

	// Copy log file to tmp so server can access it
	tmpPath := fmt.Sprintf("/tmp/pinkpanel-log-dl-%d.log", time.Now().UnixNano())
	if _, err := h.AgentClient.Call("file_copy", map[string]any{"source": logPath, "dest": tmpPath}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to read log file: " + err.Error()}})
	}
	if _, err := h.AgentClient.Call("set_permissions", map[string]any{"path": tmpPath, "mode": "644"}); err != nil {
		log.Error().Err(err).Msg("failed to set permissions on temp log file")
	}
	defer func() {
		h.AgentClient.Call("file_delete", map[string]any{"path": tmpPath, "recursive": false})
	}()

	fileName := fmt.Sprintf("%s.%s.log", dom.Name, logType)
	c.Set("Content-Type", "text/plain")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))

	return c.SendFile(tmpPath)
}

// SystemLogs reads system-level logs.
func (h *LogHandler) SystemLogs(c *fiber.Ctx) error {
	logType := c.Query("type", "syslog")
	lines, _ := strconv.Atoi(c.Query("lines", "100"))
	if lines <= 0 {
		lines = 100
	}
	if lines > 5000 {
		lines = 5000
	}
	filter := c.Query("filter", "")

	pathMap := map[string]string{
		"syslog": "/var/log/syslog",
		"auth":   "/var/log/auth.log",
		"nginx":  "/var/log/nginx/error.log",
		"mysql":  "/var/log/mysql/error.log",
	}

	logPath, ok := pathMap[logType]
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": fmt.Sprintf("unknown log type: %s", logType)}})
	}

	resp, err := h.AgentClient.Call("log_read", map[string]any{
		"path":   logPath,
		"lines":  lines,
		"filter": filter,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to read log: " + err.Error()}})
	}

	return c.JSON(fiber.Map{
		"log_type": logType,
		"path":     logPath,
		"content":  resp.Result,
	})
}
