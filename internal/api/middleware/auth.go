package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/pinkpanel/pinkpanel/internal/auth"
)

// Auth validates JWT access tokens and injects claims into the request context.
func Auth(jwtManager *auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Missing authorization header",
				},
			})
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
			})
		}

		claims, err := jwtManager.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
			})
		}

		c.Locals("admin_id", claims.AdminID)
		c.Locals("username", claims.Username)
		return c.Next()
	}
}
