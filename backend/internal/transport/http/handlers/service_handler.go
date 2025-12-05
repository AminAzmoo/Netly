package handlers

import (
    "strconv"

    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/infrastructure/logger"
)

type ServiceHandler struct {
    service ports.ServiceService
    logger  *logger.Logger
}

func NewServiceHandler(service ports.ServiceService, logger *logger.Logger) *ServiceHandler {
    return &ServiceHandler{service: service, logger: logger}
}

func (h *ServiceHandler) CreateService(c *fiber.Ctx) error {
    var input ports.CreateServiceInput
    if err := c.BodyParser(&input); err != nil {
        h.logger.Warnw("service_create_body_parse_failed", "error", err)
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
    }

    h.logger.Infow("service_create_request", "node_id", input.NodeID, "port", input.ListenPort, "protocol", input.Protocol)
    service, err := h.service.CreateService(c.Context(), input)
    if err != nil {
        h.logger.Errorw("service_create_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    h.logger.Infow("service_create_success", "id", service.ID)
    return c.Status(fiber.StatusCreated).JSON(service)
}

func (h *ServiceHandler) GetServices(c *fiber.Ctx) error {
    h.logger.Infow("service_list_request")
    services, err := h.service.GetServices(c.Context())
    if err != nil {
        h.logger.Errorw("service_list_failed", "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    h.logger.Infow("service_list_success", "count", len(services))
    return c.JSON(services)
}

func (h *ServiceHandler) GetService(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        h.logger.Warnw("service_get_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
    }

    h.logger.Infow("service_get_request", "id", id)
    service, err := h.service.GetServiceByID(c.Context(), uint(id))
    if err != nil {
        h.logger.Warnw("service_get_not_found", "id", id)
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
    }

    return c.JSON(service)
}

func (h *ServiceHandler) DeleteService(c *fiber.Ctx) error {
    id, err := strconv.Atoi(c.Params("id"))
    if err != nil {
        h.logger.Warnw("service_delete_invalid_id")
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID"})
    }

    h.logger.Infow("service_delete_request", "id", id)
    if err := h.service.DeleteService(c.Context(), uint(id)); err != nil {
        h.logger.Errorw("service_delete_failed", "id", id, "error", err)
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    h.logger.Infow("service_delete_success", "id", id)
    return c.SendStatus(fiber.StatusNoContent)
}
