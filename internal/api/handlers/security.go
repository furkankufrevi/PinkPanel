package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/agent"
)

type SecurityHandler struct {
	DB          *sql.DB
	AgentClient *agent.Client
}

// Fail2banStatus returns the overall fail2ban status and jail list.
func (h *SecurityHandler) Fail2banStatus(c *fiber.Ctx) error {
	resp, err := h.AgentClient.Call("fail2ban_status", nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AGENT_ERROR",
				"message": "Failed to get fail2ban status: " + err.Error(),
			},
		})
	}
	return c.JSON(resp.Result)
}

// Fail2banJailStatus returns detailed status of a specific jail.
func (h *SecurityHandler) Fail2banJailStatus(c *fiber.Ctx) error {
	jail := c.Params("jail")
	if jail == "" {
		jail = "pinkpanel"
	}

	resp, err := h.AgentClient.Call("fail2ban_jail_status", map[string]any{"jail": jail})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AGENT_ERROR",
				"message": "Failed to get jail status: " + err.Error(),
			},
		})
	}
	return c.JSON(resp.Result)
}

// Fail2banBannedIPs returns the list of currently banned IPs for a jail.
func (h *SecurityHandler) Fail2banBannedIPs(c *fiber.Ctx) error {
	jail := c.Query("jail", "pinkpanel")

	resp, err := h.AgentClient.Call("fail2ban_banned_ips", map[string]any{"jail": jail})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AGENT_ERROR",
				"message": "Failed to get banned IPs: " + err.Error(),
			},
		})
	}
	return c.JSON(resp.Result)
}

// Fail2banBanIP manually bans an IP in a jail.
func (h *SecurityHandler) Fail2banBanIP(c *fiber.Ctx) error {
	var req struct {
		IP   string `json:"ip"`
		Jail string `json:"jail"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.IP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "IP address is required",
			},
		})
	}
	if req.Jail == "" {
		req.Jail = "pinkpanel"
	}

	resp, err := h.AgentClient.Call("fail2ban_ban_ip", map[string]any{"ip": req.IP, "jail": req.Jail})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AGENT_ERROR",
				"message": "Failed to ban IP: " + err.Error(),
			},
		})
	}
	return c.JSON(resp.Result)
}

// Fail2banUnbanIP removes a ban for an IP in a jail.
func (h *SecurityHandler) Fail2banUnbanIP(c *fiber.Ctx) error {
	var req struct {
		IP   string `json:"ip"`
		Jail string `json:"jail"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.IP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "IP address is required",
			},
		})
	}
	if req.Jail == "" {
		req.Jail = "pinkpanel"
	}

	resp, err := h.AgentClient.Call("fail2ban_unban_ip", map[string]any{"ip": req.IP, "jail": req.Jail})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "AGENT_ERROR",
				"message": "Failed to unban IP: " + err.Error(),
			},
		})
	}
	return c.JSON(resp.Result)
}
