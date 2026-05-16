package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Context keys used to pass authenticated user info through Fiber locals.
const (
	ContextUserIDKey   = "user_id"
	ContextUserRoleKey = "user_role"
	ContextUserEmail   = "user_email"
)

// Middleware returns a Fiber handler that validates the Bearer JWT.
// On success, it stores user_id, user_role, and user_email in c.Locals.
func Middleware(jwtMgr *JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid authorization header")
		}

		claims, err := jwtMgr.Parse(parts[1])
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(ContextUserIDKey, claims.UserID)
		c.Locals(ContextUserRoleKey, claims.Role)
		c.Locals(ContextUserEmail, claims.Email)
		return c.Next()
	}
}

// RequireRole returns a middleware that allows only users with one of the given roles.
// Must be used after Middleware().
func RequireRole(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(ContextUserRoleKey).(string)
		if _, ok := allowed[role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}
