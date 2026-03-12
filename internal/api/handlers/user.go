package handlers

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/pinkpanel/pinkpanel/internal/agent"
	"github.com/pinkpanel/pinkpanel/internal/api/middleware"
	"github.com/pinkpanel/pinkpanel/internal/core/user"
	"github.com/pinkpanel/pinkpanel/internal/db"
)

// UserHandler handles user CRUD operations.
type UserHandler struct {
	DB          *sql.DB
	UserSvc     *user.Service
	AgentClient *agent.Client
	BcryptCost  int
}

type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

// List returns all users. Admin+ only.
func (h *UserHandler) List(c *fiber.Ctx) error {
	users, err := h.UserSvc.List(c.Query("search"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list users",
			},
		})
	}

	if users == nil {
		users = []user.UserWithStats{}
	}

	return c.JSON(fiber.Map{"data": users})
}

// Get returns a single user by ID.
func (h *UserHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	u, err := h.UserSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "User not found",
			},
		})
	}

	return c.JSON(u)
}

// Create creates a new user. Super admin only.
func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req createUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Username, email, and password are required",
			},
		})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Password must be at least 8 characters",
			},
		})
	}

	if req.Role == "" {
		req.Role = "user"
	}

	// Non-super admins cannot create super_admin users
	if req.Role == "super_admin" && !middleware.IsSuperAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Only super admins can create super admin accounts",
			},
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.BcryptCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to hash password",
			},
		})
	}

	u, err := h.UserSvc.Create(req.Username, req.Email, string(hash), req.Role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
	}

	// Create Linux system user for non-super_admin users
	if req.Role != "super_admin" && u.SystemUsername != nil && *u.SystemUsername != "www-data" {
		_, err := h.AgentClient.Call("user_create", map[string]interface{}{
			"username": *u.SystemUsername,
			"home_dir": "/home/" + *u.SystemUsername,
		})
		if err != nil {
			log.Printf("WARNING: failed to create system user %s: %v", *u.SystemUsername, err)
		}
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "user_create", "user", u.ID, u.Username, c.IP())

	return c.Status(fiber.StatusCreated).JSON(u)
}

// Update modifies a user's email and/or role.
func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	var req updateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	// Only super_admin can change roles to/from super_admin
	if req.Role == "super_admin" && !middleware.IsSuperAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "FORBIDDEN",
				"message": "Only super admins can assign super admin role",
			},
		})
	}

	u, err := h.UserSvc.Update(id, req.Email, req.Role)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "user_update", "user", u.ID, u.Username, c.IP())

	return c.JSON(u)
}

// Delete removes a user.
func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	// Prevent self-deletion
	adminID, _ := c.Locals("admin_id").(int64)
	if id == adminID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Cannot delete your own account",
			},
		})
	}

	u, err := h.UserSvc.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "User not found",
			},
		})
	}

	// Delete Linux system user if applicable
	if u.SystemUsername != nil && *u.SystemUsername != "www-data" && *u.SystemUsername != "" {
		_, err := h.AgentClient.Call("user_delete", map[string]interface{}{
			"username":    *u.SystemUsername,
			"remove_home": true,
		})
		if err != nil {
			log.Printf("WARNING: failed to delete system user %s: %v", *u.SystemUsername, err)
		}
	}

	if err := h.UserSvc.Delete(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": err.Error(),
			},
		})
	}

	_ = db.LogActivity(h.DB, adminID, "user_delete", "user", u.ID, u.Username, c.IP())

	return c.JSON(fiber.Map{"message": "User deleted successfully"})
}

// Suspend suspends a user account.
func (h *UserHandler) Suspend(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	// Prevent self-suspension
	adminID, _ := c.Locals("admin_id").(int64)
	if id == adminID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Cannot suspend your own account",
			},
		})
	}

	if err := h.UserSvc.Suspend(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to suspend user",
			},
		})
	}

	_ = db.LogActivity(h.DB, adminID, "user_suspend", "user", id, "", c.IP())

	u, _ := h.UserSvc.GetByID(id)
	return c.JSON(u)
}

// Activate re-activates a suspended user.
func (h *UserHandler) Activate(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	if err := h.UserSvc.Activate(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to activate user",
			},
		})
	}

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "user_activate", "user", id, "", c.IP())

	u, _ := h.UserSvc.GetByID(id)
	return c.JSON(u)
}

// ResetPassword allows admin to reset a user's password.
func (h *UserHandler) ResetPassword(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid user ID",
			},
		})
	}

	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Password must be at least 8 characters",
			},
		})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.BcryptCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to hash password",
			},
		})
	}

	if err := h.UserSvc.UpdatePassword(id, string(hash)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update password",
			},
		})
	}

	// Invalidate all tokens for this user
	h.DB.Exec("DELETE FROM refresh_tokens WHERE admin_id = ?", id)

	adminID, _ := c.Locals("admin_id").(int64)
	_ = db.LogActivity(h.DB, adminID, "user_reset_password", "user", id, "", c.IP())

	return c.JSON(fiber.Map{"message": "Password reset successfully"})
}
