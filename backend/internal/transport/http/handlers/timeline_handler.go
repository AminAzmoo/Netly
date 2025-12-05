package handlers

import (
    "strconv"
    "github.com/gofiber/fiber/v2"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/transport/http/dto"
)

type TimelineHandler struct {
	repo ports.TimelineRepository
}

func NewTimelineHandler(repo ports.TimelineRepository) *TimelineHandler {
	return &TimelineHandler{repo: repo}
}

func (h *TimelineHandler) GetEvents(c *fiber.Ctx) error {
    rtype := c.Query("resource_type")
    ridStr := c.Query("resource_id")
    if rtype != "" && ridStr != "" {
        rid64, err := strconv.ParseUint(ridStr, 10, 32)
        if err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse{ Error: "invalid resource_id" })
        }
        events, err := h.repo.GetByResource(c.Context(), rtype, uint(rid64))
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{ Error: err.Error() })
        }
        return c.JSON(events)
    }
    events, err := h.repo.GetAll(c.Context(), 50)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse{ Error: err.Error() })
    }
    return c.JSON(events)
}
