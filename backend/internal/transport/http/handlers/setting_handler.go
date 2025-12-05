package handlers

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/core/services"
	"github.com/netly/backend/internal/infrastructure/logger"
	"github.com/netly/backend/internal/transport/http/dto"
)

type SettingHandler struct {
	service       *services.SystemSettingService
	logger        *logger.Logger
	tunnelManager *services.TunnelManager
	cloudflareAPI *services.CloudflareAPIClient
}

func NewSettingHandler(service *services.SystemSettingService, logger *logger.Logger, tunnelManager *services.TunnelManager) *SettingHandler {
	return &SettingHandler{
		service:       service,
		logger:        logger,
		tunnelManager: tunnelManager,
		cloudflareAPI: services.NewCloudflareAPIClient(logger),
	}
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

	// Check if cloudflare credentials are being updated
	hasCloudflareUpdate := false
	cfEmail, _ := req["cloudflare_email"].(string)
	cfGlobalKey, _ := req["cloudflare_global_key"].(string)
	cfAccountID, _ := req["cloudflare_account_id"].(string)
	cfTunnelName, _ := req["cloudflare_tunnel_name"].(string)
	cfPublicURL, _ := req["cloudflare_public_url"].(string)

	// If we have all required cloudflare credentials, generate a token
	if cfEmail != "" && cfGlobalKey != "" && cfAccountID != "" {
		hasCloudflareUpdate = true
		h.logger.Infow("cloudflare credentials detected, generating tunnel token")

		// Create/fetch tunnel and generate token with correct format
		tunnelInfo, err := h.cloudflareAPI.CreateOrGetTunnel(
			services.CloudflareCredentials{
				Email:     cfEmail,
				GlobalKey: cfGlobalKey,
				AccountID: cfAccountID,
			},
			cfTunnelName,
		)

		if err != nil {
			h.logger.Errorw("failed to create cloudflare tunnel", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
				Error: "failed to create cloudflare tunnel: " + err.Error(),
			})
		}

		// Store the generated token and tunnel info
		req["cloudflare_token"] = tunnelInfo.Token
		req["cloudflare_tunnel_id"] = tunnelInfo.ID
		req["cloudflare_tunnel_name"] = tunnelInfo.Name
		req["cloudflare_tunnel_secret"] = tunnelInfo.TunnelSecret

		h.logger.Infow("cloudflare token generated successfully",
			"tunnel_id", tunnelInfo.ID,
			"tunnel_name", tunnelInfo.Name)
	}

	if err := h.service.UpdateSettings(c.Context(), req); err != nil {
		h.logger.Errorw("settings_update_failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
			Error: err.Error(),
		})
	}

	// If cloudflare was updated, start/restart the tunnel
	if hasCloudflareUpdate {
		token, _ := req["cloudflare_token"].(string)
		if token != "" {
			// Stop existing tunnel if running
			if h.tunnelManager.IsRunning() {
				h.logger.Info("stopping existing tunnel before restart")
				h.tunnelManager.StopTunnel()
			}

			// Start with new token
			go func() {
				if cfPublicURL != "" {
					h.tunnelManager.StartTunnelWithURL(token, cfPublicURL)
				} else {
					h.tunnelManager.StartTunnel(token)
				}
			}()

			h.logger.Infow("cloudflare tunnel started with new token")
		}
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

func (h *SettingHandler) ClearLogs(c *fiber.Ctx) error {
	// Clear log files
	logFiles := []string{"logs/app.log", "logs/error.log", "netly.log"}
	for _, file := range logFiles {
		if err := os.Truncate(file, 0); err != nil {
			h.logger.Warnw("failed to clear log file", "file", file, "error", err)
		}
	}

	h.logger.Info("Log files cleared by admin")
	return c.JSON(dto.SuccessResponse{
		Message: "logs cleared successfully",
	})
}
