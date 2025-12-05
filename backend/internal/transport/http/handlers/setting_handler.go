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



    return c.JSON(dto.SuccessResponse{
        Message: "settings updated successfully",
    })
}

type TunnelSettingsRequest struct {
    CloudflareToken string `json:"cloudflare_token"`
    PublicURL       string `json:"public_url,omitempty"`
}

func (h *SettingHandler) UpdateTunnelSettings(c *fiber.Ctx) error {
    var req TunnelSettingsRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid request body",
        })
    }

    if req.CloudflareToken == "" {
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "cloudflare_token is required",
        })
    }

    // If public_url is provided, save it first
    if req.PublicURL != "" {
        settings := map[string]interface{}{"public_url": req.PublicURL}
        if err := h.service.UpdateSettings(c.Context(), settings); err != nil {
            h.logger.Errorw("failed to save public url", "error", err)
            return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
                Error: "failed to save public url",
            })
        }
        h.logger.Infow("manual public url saved", "url", req.PublicURL)
    }

    // Start tunnel with fixed URL flag
    if err := h.tunnelManager.StartTunnelWithURL(req.CloudflareToken, req.PublicURL); err != nil {
        h.logger.Errorw("failed to start tunnel", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: "failed to start tunnel",
        })
    }

    return c.JSON(dto.SuccessResponse{
        Message: "tunnel started successfully",
    })
}
