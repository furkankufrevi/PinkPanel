package handlers

import (
	"database/sql"
	"strconv"
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

	// Check for account lockout (10 failed attempts in last 30 minutes)
	var failCount int
	h.DB.QueryRow(
		`SELECT COUNT(*) FROM login_attempts WHERE email = ? AND success = 0 AND created_at > datetime('now', '-30 minutes')`,
		req.Username,
	).Scan(&failCount)
	if failCount >= 10 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "ACCOUNT_LOCKED",
				"message": "Too many failed login attempts. Please try again in 30 minutes.",
			},
		})
	}

	// Look up admin
	var id int64
	var passwordHash, role, status string
	var totpEnabled int
	err := h.DB.QueryRow(
		"SELECT id, password_hash, role, status, totp_enabled FROM admins WHERE username = ?",
		req.Username,
	).Scan(&id, &passwordHash, &role, &status, &totpEnabled)
	if err == sql.ErrNoRows {
		h.recordLoginAttempt(req.Username, c.IP(), false)
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

	// Check account status
	if status == "suspended" {
		h.recordLoginAttempt(req.Username, c.IP(), false)
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "ACCOUNT_SUSPENDED",
				"message": "Your account has been suspended. Contact an administrator.",
			},
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		h.recordLoginAttempt(req.Username, c.IP(), false)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid username or password",
			},
		})
	}

	// Successful login — clear failed attempts
	h.recordLoginAttempt(req.Username, c.IP(), true)

	// If 2FA is enabled, return a short-lived temp token instead of real tokens
	if totpEnabled == 1 {
		tempPair, err := h.JWTManager.GenerateTokenPair(id, req.Username, role)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to generate tokens",
				},
			})
		}
		return c.JSON(fiber.Map{
			"requires_2fa": true,
			"temp_token":   tempPair.AccessToken,
		})
	}

	// Generate tokens
	tokenPair, err := h.JWTManager.GenerateTokenPair(id, req.Username, role)
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

	// Record login session
	h.DB.Exec(
		"INSERT INTO sessions (admin_id, token_hash, ip_address, user_agent, expires_at) VALUES (?, ?, ?, ?, ?)",
		id, tokenHash, c.IP(), c.Get("User-Agent"), expiresAt,
	)

	return c.JSON(fiber.Map{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"role":          role,
	})
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
	var username, role string
	var expiresAt time.Time
	err := h.DB.QueryRow(`
		SELECT rt.admin_id, a.username, a.role, rt.expires_at
		FROM refresh_tokens rt
		JOIN admins a ON a.id = rt.admin_id
		WHERE rt.token_hash = ?
	`, tokenHash).Scan(&adminID, &username, &role, &expiresAt)
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
	tokenPair, err := h.JWTManager.GenerateTokenPair(adminID, username, role)
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

	return c.JSON(fiber.Map{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"role":          role,
	})
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

// Profile returns the current user's profile info.
func (h *AuthHandler) Profile(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	var username, email, role, createdAt string
	err := h.DB.QueryRow(
		"SELECT username, email, role, created_at FROM admins WHERE id = ?", adminID,
	).Scan(&username, &email, &role, &createdAt)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get profile",
			},
		})
	}

	return c.JSON(fiber.Map{
		"id":         adminID,
		"username":   username,
		"email":      email,
		"role":       role,
		"created_at": createdAt,
	})
}

// Session represents an active login session.
type Session struct {
	ID        int64  `json:"id"`
	AdminID   int64  `json:"admin_id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	Current   bool   `json:"current"`
}

// ListSessions returns active sessions for the authenticated user.
func (h *AuthHandler) ListSessions(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	// Clean up expired sessions
	h.DB.Exec("DELETE FROM sessions WHERE expires_at < datetime('now')")

	rows, err := h.DB.Query(
		"SELECT id, admin_id, ip_address, user_agent, created_at, expires_at FROM sessions WHERE admin_id = ? ORDER BY created_at DESC",
		adminID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list sessions",
			},
		})
	}
	defer rows.Close()

	currentIP := c.IP()
	currentUA := c.Get("User-Agent")
	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.AdminID, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt); err != nil {
			continue
		}
		s.Current = s.IPAddress == currentIP && s.UserAgent == currentUA
		sessions = append(sessions, s)
	}

	if sessions == nil {
		sessions = []Session{}
	}

	return c.JSON(fiber.Map{"data": sessions})
}

// RevokeSession deletes a specific session.
func (h *AuthHandler) RevokeSession(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)
	sessionID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid session ID",
			},
		})
	}

	// Ensure user can only revoke their own sessions
	result, err := h.DB.Exec("DELETE FROM sessions WHERE id = ? AND admin_id = ?", sessionID, adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to revoke session",
			},
		})
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Session not found",
			},
		})
	}

	return c.JSON(fiber.Map{"message": "Session revoked"})
}

func (h *AuthHandler) recordLoginAttempt(username, ip string, success bool) {
	successVal := 0
	if success {
		successVal = 1
	}
	h.DB.Exec(
		"INSERT INTO login_attempts (email, ip_address, success) VALUES (?, ?, ?)",
		username, ip, successVal,
	)
}
