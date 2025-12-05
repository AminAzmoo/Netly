package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/services"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/transport/http/dto"
)

type SettingHandler struct {
    service       *services.SystemSettingService
    logger        *logger.Logger
    tunnelManager *services.TunnelManager
}

func NewSettingHandler(service *services.SystemSettingService, logger *logger.Logger, tunnelManager *services.TunnelManager) *SettingHandler {
    return &SettingHandler{service: service, logger: logger, tunnelManager: tunnelManager}
}

func (h *SettingHandler) GetSettings(c *fiber.Ctx) error {
    h.logger.Infow("settings_get_request")
    settings, err := h.service.GetSettings(c.Context())
    if err != nil {
        h.logger.Errorw("settings_get_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }
    return c.JSON(settings)
}

func (h *SettingHandler) UpdateSettings(c *fiber.Ctx) error {
    var req map[string]interface{}
    if err := c.BodyParser(&req); err != nil {
        h.logger.Warnw("settings_update_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid request body",
        })
    }

    h.logger.Infow("settings_update_request", "keys", len(req))
    if err := h.service.UpdateSettings(c.Context(), req); err != nil {
        h.logger.Errorw("settings_update_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    if token, ok := req["cloudflare_token"].(string); ok && token != "" {
        h.logger.Infow("starting_cloudflare_tunnel", "token_length", len(token))
        go func() {
            if err := h.tunnelManager.StartTunnel(token); err != nil {
                h.logger.Errorw("tunnel_start_failed", "error", err)
            }
        }()
    }

    return c.JSON(dto.SuccessResponse{
        Message: "settings updated successfully",
    })
}
