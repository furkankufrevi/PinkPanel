package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/pinkpanel/pinkpanel/internal/auth"
)

type SetupHandler struct {
	DB         *sql.DB
	JWTManager *auth.JWTManager
	BcryptCost int
}

type setupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Status returns whether initial setup has been completed.
func (h *SetupHandler) Status(c *fiber.Ctx) error {
	var count int
	err := h.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check setup status",
			},
		})
	}

	return c.JSON(fiber.Map{
		"setup_required": count == 0,
	})
}

// CreateAdmin creates the initial admin account (only works when no admins exist).
func (h *SetupHandler) CreateAdmin(c *fiber.Ctx) error {
	// Check if admins already exist
	var count int
	if err := h.DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to check admin count",
			},
		})
	}
	if count > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "SETUP_COMPLETE",
				"message": "Initial setup has already been completed",
			},
		})
	}

	var req setupRequest
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

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), h.BcryptCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to hash password",
			},
		})
	}

	// Insert admin
	result, err := h.DB.Exec(
		"INSERT INTO admins (username, email, password_hash) VALUES (?, ?, ?)",
		req.Username, req.Email, string(hash),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to create admin account",
			},
		})
	}

	adminID, _ := result.LastInsertId()

	// Mark setup as complete
	h.DB.Exec("UPDATE settings SET value = 'true' WHERE key = 'panel.setup_complete'")

	// Generate tokens so user is logged in immediately
	tokenPair, err := h.JWTManager.GenerateTokenPair(adminID, req.Username)
	if err != nil {
		// Admin created but tokens failed — they can log in manually
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Admin created successfully, please log in",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":       "Admin created successfully",
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
	})
}
