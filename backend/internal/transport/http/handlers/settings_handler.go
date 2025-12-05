package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

type SettingsHandler struct {
	logger *logger.Logger
}

func NewSettingsHandler(logger *logger.Logger) *SettingsHandler {
	return &SettingsHandler{logger: logger}
}

type SettingsRequest struct {
	SystemName  string `json:"system_name"`
	AdminEmail  string `json:"admin_email"`
	PublicURL   string `json:"public_url"`
	Environment string `json:"environment"`
}

type SettingsResponse struct {
	SystemName  string `json:"system_name"`
	AdminEmail  string `json:"admin_email"`
	PublicURL   string `json:"public_url"`
	Environment string `json:"environment"`
}

func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	return c.JSON(SettingsResponse{
		SystemName:  viper.GetString("general.system_name"),
		AdminEmail:  viper.GetString("general.admin_email"),
		PublicURL:   viper.GetString("security.public_url"),
		Environment: viper.GetString("general.environment"),
	})
}

func (h *SettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	var req SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	viper.Set("general.system_name", req.SystemName)
	viper.Set("general.admin_email", req.AdminEmail)
	viper.Set("security.public_url", req.PublicURL)
	viper.Set("general.environment", req.Environment)

	if err := viper.WriteConfig(); err != nil {
		h.logger.Error("Failed to write config", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save settings"})
	}

	return c.JSON(fiber.Map{"message": "Settings updated successfully"})
}
