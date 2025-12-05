package db

import (
    "context"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "gorm.io/gorm"
)

type serviceRepository struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewServiceRepository(db *gorm.DB, log *logger.Logger) ports.ServiceRepository {
    return &serviceRepository{db: db, log: log}
}

func (r *serviceRepository) Create(ctx context.Context, service *domain.Service) error {
    if err := r.db.WithContext(ctx).Create(service).Error; err != nil {
        r.log.Errorw("service_repo_create_failed", "node_id", service.NodeID, "error", err)
        return err
    }
    r.log.Infow("service_repo_create_ok", "id", service.ID, "node_id", service.NodeID)
    return nil
}

func (r *serviceRepository) GetByID(ctx context.Context, id uint) (*domain.Service, error) {
    var service domain.Service
    if err := r.db.WithContext(ctx).First(&service, id).Error; err != nil {
        r.log.Errorw("service_repo_get_failed", "id", id, "error", err)
        return nil, err
    }
    return &service, nil
}

func (r *serviceRepository) GetByNodeID(ctx context.Context, nodeID uint) ([]domain.Service, error) {
    var services []domain.Service
    if err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).Find(&services).Error; err != nil {
        r.log.Errorw("service_repo_get_by_node_failed", "node_id", nodeID, "error", err)
        return nil, err
    }
    r.log.Infow("service_repo_get_by_node_ok", "node_id", nodeID, "count", len(services))
    return services, nil
}

func (r *serviceRepository) GetAll(ctx context.Context) ([]domain.Service, error) {
    var services []domain.Service
    if err := r.db.WithContext(ctx).Find(&services).Error; err != nil {
        r.log.Errorw("service_repo_list_failed", "error", err)
        return nil, err
    }
    r.log.Infow("service_repo_list_ok", "count", len(services))
    return services, nil
}

func (r *serviceRepository) Update(ctx context.Context, service *domain.Service) error {
    if err := r.db.WithContext(ctx).Save(service).Error; err != nil {
        r.log.Errorw("service_repo_update_failed", "id", service.ID, "error", err)
        return err
    }
    r.log.Infow("service_repo_update_ok", "id", service.ID)
    return nil
}

func (r *serviceRepository) Delete(ctx context.Context, id uint) error {
    if err := r.db.WithContext(ctx).Delete(&domain.Service{}, id).Error; err != nil {
        r.log.Errorw("service_repo_delete_failed", "id", id, "error", err)
        return err
    }
    r.log.Infow("service_repo_delete_ok", "id", id)
    return nil
}
