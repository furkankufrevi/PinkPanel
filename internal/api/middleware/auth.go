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

		// Backward compatibility: old tokens without role are super_admin
		role := claims.Role
		if role == "" {
			role = "super_admin"
		}

		c.Locals("admin_id", claims.AdminID)
		c.Locals("username", claims.Username)
		c.Locals("role", role)
		return c.Next()
	}
}

// RequireRole returns middleware that restricts access to the given roles.
func RequireRole(roles ...string) fiber.Handler {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if !roleSet[role] {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "FORBIDDEN",
					"message": "Insufficient permissions",
				},
			})
		}
		return c.Next()
	}
}

// IsSuperAdmin returns true if the current user has the super_admin role.
func IsSuperAdmin(c *fiber.Ctx) bool {
	role, _ := c.Locals("role").(string)
	return role == "super_admin"
}

// IsAdmin returns true if the current user has admin or super_admin role.
func IsAdmin(c *fiber.Ctx) bool {
	role, _ := c.Locals("role").(string)
	return role == "super_admin" || role == "admin"
}
