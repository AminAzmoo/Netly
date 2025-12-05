package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/core/services"
	"github.com/netly/backend/internal/infrastructure/logger"
)

type CleanupHandler struct {
	cleanupService *services.CleanupService
	nodeService    ports.NodeService
	logger         *logger.Logger
}

func NewCleanupHandler(cleanupService *services.CleanupService, nodeService ports.NodeService, logger *logger.Logger) *CleanupHandler {
	return &CleanupHandler{
		cleanupService: cleanupService,
		nodeService:    nodeService,
		logger:         logger,
	}
}

type CleanupRequestDTO struct {
	NodeID      uint   `json:"node_id"`
	Mode        string `json:"mode"`
	Force       bool   `json:"force"`
	ConfirmText string `json:"confirm_text"`
}

func (h *CleanupHandler) CleanupNode(c *fiber.Ctx) error {
	var req CleanupRequestDTO
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warnw("cleanup_body_parse_failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Mode != "soft" && req.Mode != "hard" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "mode must be 'soft' or 'hard'",
		})
	}

	if req.Mode == "hard" {
		if !req.Force {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "hard cleanup requires force=true",
			})
		}
		if req.ConfirmText != "DELETE NODE" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "hard cleanup requires confirm_text='DELETE NODE'",
			})
		}
	}

	h.logger.Infow("cleanup_request", "node_id", req.NodeID, "mode", req.Mode)

	node, err := h.nodeService.GetNodeByID(c.Context(), req.NodeID)
	if err != nil {
		h.logger.Errorw("cleanup_node_not_found", "node_id", req.NodeID, "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "node not found",
		})
	}

	cleanupReq := services.CleanupRequest{
		NodeID:      req.NodeID,
		Mode:        services.CleanupMode(req.Mode),
		Force:       req.Force,
		ConfirmText: req.ConfirmText,
	}

	go func() {
		if err := h.cleanupService.CleanupNode(c.Context(), cleanupReq, node); err != nil {
			h.logger.Errorw("cleanup_failed", "node_id", req.NodeID, "mode", req.Mode, "error", err)
		}
	}()

	h.logger.Infow("cleanup_accepted", "node_id", req.NodeID, "mode", req.Mode)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "cleanup operation accepted",
		"node_id": req.NodeID,
		"mode":    req.Mode,
	})
}

func (h *CleanupHandler) UninstallNode(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		h.logger.Warnw("uninstall_invalid_id")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid node id",
		})
	}

	var req struct {
		Force       bool   `json:"force"`
		ConfirmText string `json:"confirm_text"`
	}

	if err := c.BodyParser(&req); err != nil {
		h.logger.Warnw("uninstall_body_parse_failed", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if !req.Force || req.ConfirmText != "DELETE NODE" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "uninstall requires force=true and confirm_text='DELETE NODE'",
		})
	}

	h.logger.Infow("uninstall_request", "node_id", id)

	node, err := h.nodeService.GetNodeByID(c.Context(), uint(id))
	if err != nil {
		h.logger.Errorw("uninstall_node_not_found", "node_id", id, "error", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "node not found",
		})
	}

	cleanupReq := services.CleanupRequest{
		NodeID:      uint(id),
		Mode:        services.CleanupModeHard,
		Force:       req.Force,
		ConfirmText: req.ConfirmText,
	}

	go func() {
		if err := h.cleanupService.CleanupNode(c.Context(), cleanupReq, node); err != nil {
			h.logger.Errorw("uninstall_failed", "node_id", id, "error", err)
		}
	}()

	h.logger.Infow("uninstall_accepted", "node_id", id)
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "uninstall operation accepted",
		"node_id": id,
	})
}
