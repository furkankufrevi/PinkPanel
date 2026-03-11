package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/pinkpanel/pinkpanel/internal/auth"
)

type AuthHandler struct {
	DB         *sql.DB
	JWTManager *auth.JWTManager
	BcryptCost int
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Login validates credentials and returns access + refresh tokens.
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Username and password are required",
			},
		})
	}

	// Look up admin
	var id int64
	var passwordHash string
	err := h.DB.QueryRow(
		"SELECT id, password_hash FROM admins WHERE username = ?",
		req.Username,
	).Scan(&id, &passwordHash)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "An internal error occurred",
			},
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
	}

	// Generate tokens
	tokenPair, err := h.JWTManager.GenerateTokenPair(id, req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate tokens",
			},
		})
	}

	// Store refresh token hash in DB
	tokenHash := auth.HashToken(tokenPair.RefreshToken)
	expiresAt := time.Now().Add(h.JWTManager.RefreshTokenTTL())
	_, err = h.DB.Exec(
		"INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES (?, ?, ?)",
		id, tokenHash, expiresAt,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to store refresh token",
			},
		})
	}

	return c.JSON(tokenPair)
}

// Refresh issues a new access token from a valid refresh token.
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Refresh token is required",
			},
		})
	}

	tokenHash := auth.HashToken(req.RefreshToken)

	var adminID int64
	var username string
	var expiresAt time.Time
	err := h.DB.QueryRow(`
		SELECT rt.admin_id, a.username, rt.expires_at
		FROM refresh_tokens rt
		JOIN admins a ON a.id = rt.admin_id
		WHERE rt.token_hash = ?
	`, tokenHash).Scan(&adminID, &username, &expiresAt)
	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_TOKEN",
				"message": "Invalid refresh token",
			},
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "An internal error occurred",
			},
		})
	}

	if time.Now().After(expiresAt) {
		// Clean up expired token
		h.DB.Exec("DELETE FROM refresh_tokens WHERE token_hash = ?", tokenHash)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "TOKEN_EXPIRED",
				"message": "Refresh token has expired",
			},
		})
	}

	// Delete old refresh token (rotation)
	h.DB.Exec("DELETE FROM refresh_tokens WHERE token_hash = ?", tokenHash)

	// Generate new token pair
	tokenPair, err := h.JWTManager.GenerateTokenPair(adminID, username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate tokens",
			},
		})
	}

	// Store new refresh token
	newTokenHash := auth.HashToken(tokenPair.RefreshToken)
	newExpiresAt := time.Now().Add(h.JWTManager.RefreshTokenTTL())
	h.DB.Exec(
		"INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES (?, ?, ?)",
		adminID, newTokenHash, newExpiresAt,
	)

	return c.JSON(tokenPair)
}

// Logout invalidates the refresh token.
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.RefreshToken != "" {
		tokenHash := auth.HashToken(req.RefreshToken)
		h.DB.Exec("DELETE FROM refresh_tokens WHERE token_hash = ?", tokenHash)
	}

	// Also delete all tokens for this admin if authenticated
	adminID, ok := c.Locals("admin_id").(int64)
	if ok && adminID > 0 {
		h.DB.Exec("DELETE FROM refresh_tokens WHERE admin_id = ?", adminID)
	}

	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword updates the current admin's password.
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	adminID, ok := c.Locals("admin_id").(int64)
	if !ok || adminID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHORIZED",
				"message": "Not authenticated",
			},
		})
	}

	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Current and new passwords are required",
			},
		})
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "New password must be at least 8 characters",
			},
		})
	}

	// Verify current password
	var passwordHash string
	err := h.DB.QueryRow("SELECT password_hash FROM admins WHERE id = ?", adminID).Scan(&passwordHash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "An internal error occurred",
			},
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Current password is incorrect",
			},
		})
	}

	// Hash and update
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), h.BcryptCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to hash password",
			},
		})
	}

	_, err = h.DB.Exec("UPDATE admins SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(newHash), adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to update password",
			},
		})
	}

	return c.JSON(fiber.Map{"message": "Password changed successfully"})
}
