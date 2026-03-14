package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/core/monitor"
)

type MonitorHandler struct {
	MonitorSvc *monitor.Service
}

// SystemHistory returns system metrics for the last N hours.
func (h *MonitorHandler) SystemHistory(c *fiber.Ctx) error {
	hours, _ := strconv.Atoi(c.Query("hours", "24"))

	snapshots, err := h.MonitorSvc.GetSystemHistory(hours)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if snapshots == nil {
		snapshots = []monitor.SystemSnapshot{}
	}
	return c.JSON(fiber.Map{"data": snapshots})
}

// SystemCurrent returns the latest system snapshot.
func (h *MonitorHandler) SystemCurrent(c *fiber.Ctx) error {
	snap, err := h.MonitorSvc.GetSystemCurrent()
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "no metrics collected yet"}})
	}
	return c.JSON(snap)
}

// DomainMetrics returns metrics for a specific domain.
func (h *MonitorHandler) DomainMetrics(c *fiber.Ctx) error {
	domainID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid domain ID"}})
	}

	hours, _ := strconv.Atoi(c.Query("hours", "168"))

	history, err := h.MonitorSvc.GetDomainMetrics(domainID, hours)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if history == nil {
		history = []monitor.DomainSnapshot{}
	}

	latest, _ := h.MonitorSvc.GetDomainLatest(domainID)

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"current": latest,
			"history": history,
		},
	})
}
