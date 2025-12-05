package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/config"
)

func AdminAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := cfg.Auth.AdminAPIKey
		if apiKey == "" {
			return c.Next()
		}

		headerToken := c.Get("X-Admin-Token")
		if headerToken == "" {
			auth := c.Get("Authorization")
			const prefix = "Bearer "
			if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
				headerToken = auth[len(prefix):]
			}
		}

		if headerToken != apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		return c.Next()
	}
}

func AgentAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := cfg.Auth.AgentToken
		if token == "" {
			return c.Next()
		}

		headerToken := c.Get("X-Agent-Token")
		if headerToken == "" {
			auth := c.Get("Authorization")
			const prefix = "Bearer "
			if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
				headerToken = auth[len(prefix):]
			}
		}

		if headerToken != token {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		return c.Next()
	}
}
