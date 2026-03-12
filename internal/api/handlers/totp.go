package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/pinkpanel/pinkpanel/internal/auth"
)

type TOTPHandler struct {
	DB         *sql.DB
	JWTManager *auth.JWTManager
	BcryptCost int
}

// Setup generates a TOTP secret and returns a QR code for the user to scan.
// Does not enable 2FA yet — the user must verify with Enable.
func (h *TOTPHandler) Setup(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)
	username, _ := c.Locals("username").(string)

	// Check if already enabled
	var enabled int
	h.DB.QueryRow("SELECT totp_enabled FROM admins WHERE id = ?", adminID).Scan(&enabled)
	if enabled == 1 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "ALREADY_ENABLED",
				"message": "Two-factor authentication is already enabled",
			},
		})
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "PinkPanel",
		AccountName: username,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate TOTP secret",
			},
		})
	}

	// Store secret (not yet enabled)
	_, err = h.DB.Exec("UPDATE admins SET totp_secret = ? WHERE id = ?", key.Secret(), adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to save TOTP secret",
			},
		})
	}

	// Generate QR code image
	img, err := key.Image(200, 200)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate QR code",
			},
		})
	}

	// Encode as base64 PNG
	var buf []byte
	buf, err = encodePNG(img)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to encode QR code",
			},
		})
	}
	qrBase64 := base64.StdEncoding.EncodeToString(buf)

	return c.JSON(fiber.Map{
		"secret":   key.Secret(),
		"qr_code":  "data:image/png;base64," + qrBase64,
		"otpauth":  key.URL(),
	})
}

// Enable verifies a TOTP code and enables 2FA. Also generates recovery codes.
func (h *TOTPHandler) Enable(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil || req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Verification code is required",
			},
		})
	}

	// Get stored secret
	var secret sql.NullString
	var enabled int
	h.DB.QueryRow("SELECT totp_secret, totp_enabled FROM admins WHERE id = ?", adminID).Scan(&secret, &enabled)

	if !secret.Valid || secret.String == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_SETUP",
				"message": "TOTP has not been set up. Call setup first.",
			},
		})
	}
	if enabled == 1 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "ALREADY_ENABLED",
				"message": "Two-factor authentication is already enabled",
			},
		})
	}

	// Validate code
	if !totp.Validate(req.Code, secret.String) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CODE",
				"message": "Invalid verification code",
			},
		})
	}

	// Enable 2FA
	_, err := h.DB.Exec("UPDATE admins SET totp_enabled = 1 WHERE id = ?", adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to enable 2FA",
			},
		})
	}

	// Generate recovery codes
	codes, err := h.generateRecoveryCodes(adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate recovery codes",
			},
		})
	}

	return c.JSON(fiber.Map{
		"message":        "Two-factor authentication enabled",
		"recovery_codes": codes,
	})
}

// Disable turns off 2FA after password confirmation.
func (h *TOTPHandler) Disable(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Password is required to disable 2FA",
			},
		})
	}

	// Verify password
	var passwordHash string
	h.DB.QueryRow("SELECT password_hash FROM admins WHERE id = ?", adminID).Scan(&passwordHash)
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Incorrect password",
			},
		})
	}

	// Disable 2FA and clear secret
	h.DB.Exec("UPDATE admins SET totp_enabled = 0, totp_secret = NULL WHERE id = ?", adminID)
	h.DB.Exec("DELETE FROM recovery_codes WHERE admin_id = ?", adminID)

	return c.JSON(fiber.Map{"message": "Two-factor authentication disabled"})
}

// Status returns whether 2FA is enabled for the current user.
func (h *TOTPHandler) Status(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	var enabled int
	h.DB.QueryRow("SELECT totp_enabled FROM admins WHERE id = ?", adminID).Scan(&enabled)

	// Count remaining recovery codes
	var remaining int
	h.DB.QueryRow("SELECT COUNT(*) FROM recovery_codes WHERE admin_id = ? AND used = 0", adminID).Scan(&remaining)

	return c.JSON(fiber.Map{
		"enabled":          enabled == 1,
		"recovery_remaining": remaining,
	})
}

// RegenerateRecoveryCodes generates new recovery codes (invalidates old ones).
func (h *TOTPHandler) RegenerateRecoveryCodes(c *fiber.Ctx) error {
	adminID, _ := c.Locals("admin_id").(int64)

	var enabled int
	h.DB.QueryRow("SELECT totp_enabled FROM admins WHERE id = ?", adminID).Scan(&enabled)
	if enabled != 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_ENABLED",
				"message": "Two-factor authentication is not enabled",
			},
		})
	}

	codes, err := h.generateRecoveryCodes(adminID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate recovery codes",
			},
		})
	}

	return c.JSON(fiber.Map{"recovery_codes": codes})
}

// Verify handles the 2FA step during login.
// Expects a temp_token (short-lived JWT) + TOTP code or recovery code.
func (h *TOTPHandler) Verify(c *fiber.Ctx) error {
	var req struct {
		TempToken    string `json:"temp_token"`
		Code         string `json:"code"`
		RecoveryCode string `json:"recovery_code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Invalid request body",
			},
		})
	}

	if req.TempToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Temporary token is required",
			},
		})
	}

	if req.Code == "" && req.RecoveryCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "TOTP code or recovery code is required",
			},
		})
	}

	// Validate temp token
	claims, err := h.JWTManager.ValidateAccessToken(req.TempToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_TOKEN",
				"message": "Invalid or expired temporary token",
			},
		})
	}

	// Get TOTP secret
	var secret string
	err = h.DB.QueryRow("SELECT totp_secret FROM admins WHERE id = ? AND totp_enabled = 1", claims.AdminID).Scan(&secret)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_TOKEN",
				"message": "Invalid account state",
			},
		})
	}

	verified := false

	if req.Code != "" {
		// Verify TOTP code
		verified = totp.Validate(req.Code, secret)
	} else if req.RecoveryCode != "" {
		// Verify recovery code
		verified = h.useRecoveryCode(claims.AdminID, req.RecoveryCode)
	}

	if !verified {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_CODE",
				"message": "Invalid verification code",
			},
		})
	}

	// Generate real token pair
	tokenPair, err := h.JWTManager.GenerateTokenPair(claims.AdminID, claims.Username, claims.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to generate tokens",
			},
		})
	}

	// Store refresh token
	tokenHash := auth.HashToken(tokenPair.RefreshToken)
	expiresAt := time.Now().Add(h.JWTManager.RefreshTokenTTL())
	h.DB.Exec(
		"INSERT INTO refresh_tokens (admin_id, token_hash, expires_at) VALUES (?, ?, ?)",
		claims.AdminID, tokenHash, expiresAt,
	)
	h.DB.Exec(
		"INSERT INTO sessions (admin_id, token_hash, ip_address, user_agent, expires_at) VALUES (?, ?, ?, ?, ?)",
		claims.AdminID, tokenHash, c.IP(), c.Get("User-Agent"), expiresAt,
	)

	return c.JSON(fiber.Map{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"role":          claims.Role,
	})
}

// generateRecoveryCodes creates 10 one-time recovery codes.
func (h *TOTPHandler) generateRecoveryCodes(adminID int64) ([]string, error) {
	// Delete existing codes
	h.DB.Exec("DELETE FROM recovery_codes WHERE admin_id = ?", adminID)

	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code

		// Store hashed
		hash := sha256Hash(code)
		h.DB.Exec(
			"INSERT INTO recovery_codes (admin_id, code_hash) VALUES (?, ?)",
			adminID, hash,
		)
	}
	return codes, nil
}

// useRecoveryCode checks and marks a recovery code as used.
func (h *TOTPHandler) useRecoveryCode(adminID int64, code string) bool {
	hash := sha256Hash(code)
	result, err := h.DB.Exec(
		"UPDATE recovery_codes SET used = 1 WHERE admin_id = ? AND code_hash = ? AND used = 0",
		adminID, hash,
	)
	if err != nil {
		return false
	}
	rows, _ := result.RowsAffected()
	return rows > 0
}

func generateRecoveryCode() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := fmt.Sprintf("%x", b) // 10 hex chars
	// Format as xxxxx-xxxxx
	return code[:5] + "-" + code[5:], nil
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
