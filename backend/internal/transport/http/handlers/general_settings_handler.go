package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/infrastructure/logger"
	"github.com/spf13/viper"
)

type GeneralSettingsHandler struct {
	logger *logger.Logger
}

func NewGeneralSettingsHandler(logger *logger.Logger) *GeneralSettingsHandler {
	return &GeneralSettingsHandler{logger: logger}
}

type GeneralSettingsRequest struct {
	SystemName  string `json:"systemName"`
	AdminEmail  string `json:"adminEmail"`
	PublicURL   string `json:"publicUrl"`
	Environment string `json:"environment"`
}

type GeneralSettingsResponse struct {
	SystemName  string `json:"systemName"`
	AdminEmail  string `json:"adminEmail"`
	PublicURL   string `json:"publicUrl"`
	Environment string `json:"environment"`
}

func (h *GeneralSettingsHandler) GetSettings(c *fiber.Ctx) error {
	publicURL := viper.GetString("security.public_url")
	if publicURL == "" {
		publicURL = "https://YOUR-TUNNEL-URL.trycloudflare.com"
	}
	return c.JSON(GeneralSettingsResponse{
		SystemName:  viper.GetString("general.system_name"),
		AdminEmail:  viper.GetString("general.admin_email"),
		PublicURL:   publicURL,
		Environment: viper.GetString("general.environment"),
	})
}

func (h *GeneralSettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	var req GeneralSettingsRequest
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
