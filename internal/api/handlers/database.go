package handlers

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	dbpkg "github.com/pinkpanel/pinkpanel/internal/core/database"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

type DatabaseHandler struct {
	DB          *sql.DB
	DBSvc       *dbpkg.Service
	AgentClient *agent.Client
}

// List returns all databases.
func (h *DatabaseHandler) List(c *fiber.Ctx) error {
	var domainID *int64
	if did := c.Query("domain_id"); did != "" {
		id, err := strconv.ParseInt(did, 10, 64)
		if err == nil {
			domainID = &id
		}
	}

	dbs, err := h.DBSvc.List(domainID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}
	if dbs == nil {
		dbs = []dbpkg.Database{}
	}
	return c.JSON(fiber.Map{"data": dbs})
}

// Get returns a database with its users.
func (h *DatabaseHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid database ID"}})
	}
	d, err := h.DBSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}
	users, err := h.DBSvc.ListUsers(id)
	if err != nil {
		users = []dbpkg.DatabaseUser{}
	}
	if users == nil {
		users = []dbpkg.DatabaseUser{}
	}
	return c.JSON(fiber.Map{"database": d, "users": users})
}

// Create creates a new database.
func (h *DatabaseHandler) Create(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"`
		DomainID *int64 `json:"domain_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "name is required"}})
	}

	// Create in MySQL via agent
	if _, err := h.AgentClient.Call("mysql_create_db", map[string]any{
		"name": req.Name,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to create MySQL database: " + err.Error()}})
	}

	// Store in panel database
	d, err := h.DBSvc.Create(req.Name, req.DomainID)
	if err != nil {
		// Rollback: drop the MySQL database
		if _, rollbackErr := h.AgentClient.Call("mysql_drop_db", map[string]any{"name": req.Name}); rollbackErr != nil {
			log.Error().Err(rollbackErr).Msg("failed to rollback MySQL database creation")
		}
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_database", "database", d.ID, req.Name, c.IP())

	return c.Status(201).JSON(d)
}

// Delete removes a database.
func (h *DatabaseHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid database ID"}})
	}

	d, err := h.DBSvc.GetByID(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Drop all users for this database first
	users, _ := h.DBSvc.ListUsers(id)
	for _, u := range users {
		if _, err := h.AgentClient.Call("mysql_drop_user", map[string]any{
			"username": u.Username, "host": u.Host,
		}); err != nil {
			log.Error().Err(err).Str("username", u.Username).Msg("failed to drop MySQL user")
		}
	}

	// Drop MySQL database via agent
	if _, err := h.AgentClient.Call("mysql_drop_db", map[string]any{
		"name": d.Name,
	}); err != nil {
		log.Error().Err(err).Str("database", d.Name).Msg("failed to drop MySQL database")
	}

	// Remove from panel database
	if err := h.DBSvc.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "internal_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_database", "database", id, d.Name, c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}

// CreateUser creates a database user.
func (h *DatabaseHandler) CreateUser(c *fiber.Ctx) error {
	dbID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid database ID"}})
	}

	d, err := h.DBSvc.GetByID(dbID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": "database not found"}})
	}

	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Host        string `json:"host"`
		Permissions string `json:"permissions"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid request body"}})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": "username and password are required"}})
	}
	if req.Host == "" {
		req.Host = "localhost"
	}
	if req.Permissions == "" {
		req.Permissions = "ALL"
	}

	// Create MySQL user via agent
	if _, err := h.AgentClient.Call("mysql_create_user", map[string]any{
		"username": req.Username, "password": req.Password, "host": req.Host,
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": fiber.Map{"code": "agent_error", "message": "failed to create MySQL user: " + err.Error()}})
	}

	// Grant permissions
	if _, err := h.AgentClient.Call("mysql_grant", map[string]any{
		"username": req.Username, "host": req.Host, "database": d.Name, "permissions": req.Permissions,
	}); err != nil {
		log.Error().Err(err).Msg("failed to grant permissions")
	}

	// Store in panel database
	u, err := h.DBSvc.CreateUser(dbID, req.Username, req.Host, req.Permissions)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "validation_error", "message": err.Error()}})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "create_db_user", "database", dbID, fmt.Sprintf("%s@%s", req.Username, req.Host), c.IP())

	return c.Status(201).JSON(u)
}

// DeleteUser removes a database user.
func (h *DatabaseHandler) DeleteUser(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("userId"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fiber.Map{"code": "bad_request", "message": "invalid user ID"}})
	}

	u, err := h.DBSvc.DeleteUser(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": fiber.Map{"code": "not_found", "message": err.Error()}})
	}

	// Drop MySQL user via agent
	if _, err := h.AgentClient.Call("mysql_drop_user", map[string]any{
		"username": u.Username, "host": u.Host,
	}); err != nil {
		log.Error().Err(err).Str("username", u.Username).Msg("failed to drop MySQL user via agent")
	}

	adminID, _ := c.Locals("admin_id").(int64)
	db.LogActivity(h.DB, adminID, "delete_db_user", "database", u.DatabaseID, fmt.Sprintf("%s@%s", u.Username, u.Host), c.IP())

	return c.JSON(fiber.Map{"status": "ok"})
}
