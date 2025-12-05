package db

import (
	"context"

	"github.com/netly/backend/internal/core/ports"
	"github.com/netly/backend/internal/domain"
	"github.com/netly/backend/internal/infrastructure/logger"
)

// TimelineRepoStub is a stub implementation for development
type TimelineRepoStub struct {
	logger *logger.Logger
}

func NewTimelineRepoStub(log *logger.Logger) ports.TimelineRepository {
	return &TimelineRepoStub{logger: log}
}

func (r *TimelineRepoStub) Create(ctx context.Context, event *domain.TimelineEvent) error {
	r.logger.Infow("timeline event",
		"type", event.Type,
		"status", event.Status,
		"message", event.Message,
		"resource_type", event.ResourceType,
		"resource_id", event.ResourceID,
	)
	return nil
}

func (r *TimelineRepoStub) GetByID(ctx context.Context, id uint) (*domain.TimelineEvent, error) {
	return nil, nil
}

func (r *TimelineRepoStub) GetByResource(ctx context.Context, resourceType string, resourceID uint) ([]domain.TimelineEvent, error) {
	return nil, nil
}

func (r *TimelineRepoStub) GetAll(ctx context.Context, limit int) ([]domain.TimelineEvent, error) {
	return nil, nil
}

func (r *TimelineRepoStub) Update(ctx context.Context, event *domain.TimelineEvent) error {
	return nil
}
