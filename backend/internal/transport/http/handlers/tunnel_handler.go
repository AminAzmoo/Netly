package handlers

import (
    "strconv"

    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/transport/http/dto"
)

type TunnelHandler struct {
    service ports.TunnelService
    logger  *logger.Logger
}

func NewTunnelHandler(service ports.TunnelService, logger *logger.Logger) *TunnelHandler {
    return &TunnelHandler{service: service, logger: logger}
}

func (h *TunnelHandler) CreateTunnel(c *fiber.Ctx) error {
    var req struct {
        Name         string               `json:"name"`
        Protocol     domain.TunnelProtocol `json:"protocol"`
        SourceNodeID uint                 `json:"source_node_id"`
        DestNodeID   uint                 `json:"dest_node_id"`
    }

    if err := c.BodyParser(&req); err != nil {
        h.logger.Warnw("tunnel_create_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid request body",
        })
    }

	input := ports.CreateTunnelInput{
		Name:         req.Name,
		Protocol:     req.Protocol,
		SourceNodeID: req.SourceNodeID,
		DestNodeID:   req.DestNodeID,
	}

    h.logger.Infow("tunnel_create_request", "source_node_id", req.SourceNodeID, "dest_node_id", req.DestNodeID, "protocol", req.Protocol)
    tunnel, err := h.service.CreateTunnel(c.Context(), input)
    if err != nil {
        h.logger.Errorw("tunnel_create_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    h.logger.Infow("tunnel_create_success", "id", tunnel.ID)
    return c.Status(fiber.StatusCreated).JSON(tunnel)
}

func (h *TunnelHandler) CreateChainTunnel(c *fiber.Ctx) error {
    var req struct {
        Type     string                 `json:"type"`
        Nodes    []uint                 `json:"nodes"`
        Protocol domain.TunnelProtocol `json:"protocol"`
    }

    if err := c.BodyParser(&req); err != nil {
        h.logger.Warnw("tunnel_chain_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{ Error: "invalid request body" })
    }

    if req.Type != "chain" || len(req.Nodes) < 3 {
        h.logger.Warnw("tunnel_chain_invalid_payload")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{ Error: "invalid chain payload" })
    }

    entryID := req.Nodes[0]
    relayID := req.Nodes[1]
    exitID := req.Nodes[2]

    h.logger.Infow("tunnel_chain_create_request", "entry_id", entryID, "relay_id", relayID, "exit_id", exitID, "protocol", req.Protocol)
    tunnel, err := h.service.CreateChain(c.Context(), entryID, relayID, exitID, req.Protocol)
    if err != nil {
        h.logger.Errorw("tunnel_chain_create_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{ Error: err.Error() })
    }

    h.logger.Infow("tunnel_chain_create_success", "id", tunnel.ID)
    return c.Status(fiber.StatusCreated).JSON(tunnel)
}

func (h *TunnelHandler) GetTunnels(c *fiber.Ctx) error {
    h.logger.Infow("tunnel_list_request")
    tunnels, err := h.service.GetTunnels(c.Context())
    if err != nil {
        h.logger.Errorw("tunnel_list_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    h.logger.Infow("tunnel_list_success", "count", len(tunnels))
    return c.JSON(tunnels)
}

func (h *TunnelHandler) GetTunnel(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        h.logger.Warnw("tunnel_get_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid tunnel id",
        })
    }

    h.logger.Infow("tunnel_get_request", "id", id)
    tunnel, err := h.service.GetTunnelByID(c.Context(), uint(id))
    if err != nil {
        h.logger.Warnw("tunnel_get_not_found", "id", id)
        return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
            Error: "tunnel not found",
        })
    }

    return c.JSON(tunnel)
}

func (h *TunnelHandler) DeleteTunnel(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        h.logger.Warnw("tunnel_delete_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid tunnel id",
        })
    }

    h.logger.Infow("tunnel_delete_request", "id", id)
    if err := h.service.DeleteTunnel(c.Context(), uint(id)); err != nil {
        h.logger.Warnw("tunnel_delete_failed", "id", id, "error", err)
        return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
            Error: "tunnel not found",
        })
    }

    h.logger.Infow("tunnel_delete_success", "id", id)
    return c.JSON(dto.SuccessResponse{
        Message: "tunnel deleted successfully",
    })
}
