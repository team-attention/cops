package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/team-attention/cops/api/internal/platform/util/jwtutil"
)

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	UserIDContextKey contextKey = "userId"
)

// AuthMiddleware creates a Fiber middleware for JWT authentication.
func AuthMiddleware(l *slog.Logger, jwtCfg *jwtutil.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		userID, err := jwtutil.ValidateAccessToken(jwtCfg, tokenString)
		if err != nil {
			l.Warn("invalid access token",
				slog.String("error", err.Error()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		c.Locals(UserIDContextKey, userID)

		return c.Next()
	}
}

// GetUserID extracts userID from Fiber context.
func GetUserID(c *fiber.Ctx) string {
	value := c.Locals(UserIDContextKey)
	if value == nil {
		return ""
	}

	userID, ok := value.(string)
	if !ok {
		return ""
	}

	return userID
}
