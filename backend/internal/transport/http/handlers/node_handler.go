package handlers

import (
    "strconv"

    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/core/services"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/transport/http/dto"
)

type NodeHandler struct {
    service ports.NodeService
    logger  *logger.Logger
}

func NewNodeHandler(service ports.NodeService, logger *logger.Logger) *NodeHandler {
    return &NodeHandler{service: service, logger: logger}
}

func (h *NodeHandler) CreateNode(c *fiber.Ctx) error {
    var req dto.CreateNodeRequest
    if err := c.BodyParser(&req); err != nil {
        h.logger.Warnw("node_create_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid request body",
        })
    }

    if errors := req.Validate(); len(errors) > 0 {
        h.logger.Warnw("node_create_validation_failed", "details", errors)
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error:   "validation failed",
            Details: errors,
        })
    }

	input := ports.CreateNodeInput{
		Name:     req.Name,
		IP:       req.IP,
		SSHPort:  req.GetSSHPort(),
		Role:     req.GetRole(),
		User:     req.Username,
		Password: req.Password,
		SSHKey:   req.PrivateKey,
		GeoData:  req.GeoData,
	}

    h.logger.Infow("node_create_request", "name", req.Name, "ip", req.IP, "role", req.Role)
    node, err := h.service.CreateNode(c.Context(), input)
    if err != nil {
        if err == services.ErrNodeAlreadyExists {
            h.logger.Warnw("node_create_conflict", "ip", req.IP)
            return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse{
                Error: "node with this ip already exists",
            })
        }
        if err == services.ErrNodeInvalidIP || err == services.ErrNodeInvalidInput || err == services.ErrNodeBlacklistedIP {
            h.logger.Warnw("node_create_bad_request", "ip", req.IP, "error", err)
            return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
                Error: err.Error(),
            })
        }
        h.logger.Errorw("node_create_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    h.logger.Infow("node_create_success", "id", node.ID, "ip", node.IP)
    return c.Status(fiber.StatusCreated).JSON(dto.NodeToResponse(node))
}

func (h *NodeHandler) GetNodes(c *fiber.Ctx) error {
    h.logger.Infow("nodes_list_request")
    nodes, err := h.service.GetNodes(c.Context())
    if err != nil {
        h.logger.Errorw("nodes_list_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    h.logger.Infow("nodes_list_success", "count", len(nodes))
    return c.JSON(dto.NodesToResponse(nodes))
}

func (h *NodeHandler) GetNode(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        h.logger.Warnw("node_get_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid node id",
        })
    }

    h.logger.Infow("node_get_request", "id", id)
    node, err := h.service.GetNodeByID(c.Context(), uint(id))
    if err != nil {
        h.logger.Warnw("node_get_not_found", "id", id)
        return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
            Error: "node not found",
        })
    }

    return c.JSON(dto.NodeToResponse(node))
}

func (h *NodeHandler) UpdateNode(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid node id",
        })
    }

    var req dto.UpdateNodeRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid request body",
        })
    }

    input := ports.UpdateNodeInput{
        Name:       req.Name,
        SSHPort:    req.SSHPort,
        Role:       req.Role,
        Username:   req.Username,
        Password:   req.Password,
        PrivateKey: req.PrivateKey,
    }

    node, err := h.service.UpdateNode(c.Context(), uint(id), input)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{
            Error: err.Error(),
        })
    }

    return c.JSON(dto.NodeToResponse(node))
}

func (h *NodeHandler) DeleteNode(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        h.logger.Warnw("node_delete_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid node id",
        })
    }

    h.logger.Infow("node_delete_request", "id", id)
    if err := h.service.DeleteNode(c.Context(), uint(id)); err != nil {
        h.logger.Warnw("node_delete_failed", "id", id, "error", err)
        return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
            Error: "node not found",
        })
    }

    h.logger.Infow("node_delete_success", "id", id)
    return c.JSON(dto.SuccessResponse{
        Message: "node deleted successfully",
    })
}

// InstallAgent initiates the installation and returns a Task ID (Async)
func (h *NodeHandler) InstallAgent(c *fiber.Ctx) error {
    id, err := strconv.ParseUint(c.Params("id"), 10, 32)
    if err != nil {
        h.logger.Warnw("agent_install_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "invalid node id",
        })
    }

    h.logger.Infow("agent_install_request", "id", id)
    taskID, err := h.service.InstallAgentAsync(c.Context(), uint(id))
    if err != nil {
        if err == services.ErrNodeBlacklistedIP || err == services.ErrNodeInvalidIP {
            h.logger.Warnw("agent_install_bad_request", "id", id, "error", err)
            return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{ Error: err.Error() })
        }
        h.logger.Errorw("agent_install_failed", "id", id, "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{ Error: err.Error() })
    }

    h.logger.Infow("agent_install_started", "id", id, "task_id", taskID)
    return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
        "message": "agent installation started",
        "task_id": taskID,
    })
}

// GetTaskStatus retrieves the status of an async task
func (h *NodeHandler) GetNodeCommand(c *fiber.Ctx) error {
	nodeID := c.Params("id")
	if nodeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{Error: "Missing node ID"})
	}

	// This will be handled by InstallHandler, just redirect
	return c.Redirect("/api/v1/nodes/" + nodeID + "/install-command")
}

func (h *NodeHandler) GetTaskStatus(c *fiber.Ctx) error {
    taskID := c.Params("id")
    if taskID == "" {
        h.logger.Warnw("task_status_missing_id")
        return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{
            Error: "task id is required",
        })
    }

    h.logger.Infow("task_status_request", "task_id", taskID)
    task, err := h.service.GetTaskStatus(taskID)
    if err != nil {
        h.logger.Warnw("task_status_not_found", "task_id", taskID)
        return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse{
            Error: "task not found",
        })
    }

    return c.JSON(task)
}
