package db

import (
    "context"
    "time"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "gorm.io/gorm"
)

type timelineRepository struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewTimelineRepository(db *gorm.DB, log *logger.Logger) ports.TimelineRepository {
    return &timelineRepository{
        db:  db,
        log: log,
    }
}

func (r *timelineRepository) Create(ctx context.Context, event *domain.TimelineEvent) error {
    if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
        r.log.Errorw("timeline_repo_create_failed", "type", event.Type, "status", event.Status, "error", err)
        return err
    }
    r.log.Infow("timeline_repo_create_ok", "id", event.ID, "type", event.Type, "status", event.Status)
    return nil
}

func (r *timelineRepository) GetAll(ctx context.Context, limit int) ([]domain.TimelineEvent, error) {
    var events []domain.TimelineEvent
    err := r.db.WithContext(ctx).
        Order("created_at desc").
        Limit(limit).
        Find(&events).Error
    if err != nil {
        r.log.Errorw("timeline_repo_list_failed", "error", err)
        return nil, err
    }
    r.log.Infow("timeline_repo_list_ok", "count", len(events))
    return events, nil
}

func (r *timelineRepository) GetByResource(ctx context.Context, resourceType string, resourceID uint) ([]domain.TimelineEvent, error) {
    var events []domain.TimelineEvent
    err := r.db.WithContext(ctx).
        Where("resource_type = ? AND resource_id = ?", resourceType, resourceID).
        Order("created_at desc").
        Limit(50).
        Find(&events).Error
    if err != nil {
        r.log.Errorw("timeline_repo_get_by_resource_failed", "resource_type", resourceType, "resource_id", resourceID, "error", err)
        return nil, err
    }
    r.log.Infow("timeline_repo_get_by_resource_ok", "resource_type", resourceType, "resource_id", resourceID, "count", len(events))
    return events, nil
}

func (r *timelineRepository) GetByID(ctx context.Context, id uint) (*domain.TimelineEvent, error) {
    var event domain.TimelineEvent
    err := r.db.WithContext(ctx).First(&event, id).Error
    if err != nil {
        r.log.Errorw("timeline_repo_get_failed", "id", id, "error", err)
        return nil, err
    }
    return &event, nil
}

func (r *timelineRepository) Update(ctx context.Context, event *domain.TimelineEvent) error {
    if err := r.db.WithContext(ctx).Save(event).Error; err != nil {
        r.log.Errorw("timeline_repo_update_failed", "id", event.ID, "error", err)
        return err
    }
    r.log.Infow("timeline_repo_update_ok", "id", event.ID)
    return nil
}

// CleanupOld removes events older than the specified duration
func (r *timelineRepository) CleanupOld(ctx context.Context, olderThan time.Duration) error {
    cutoff := time.Now().Add(-olderThan)
    if err := r.db.WithContext(ctx).
        Where("created_at < ?", cutoff).
        Delete(&domain.TimelineEvent{}).Error; err != nil {
        r.log.Errorw("timeline_repo_cleanup_failed", "error", err)
        return err
    }
    r.log.Infow("timeline_repo_cleanup_ok")
    return nil
}
