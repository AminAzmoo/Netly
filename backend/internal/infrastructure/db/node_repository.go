package db

import (
    "context"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "gorm.io/gorm"
)

type nodeRepository struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewNodeRepository(db *gorm.DB, log *logger.Logger) ports.NodeRepository {
    return &nodeRepository{db: db, log: log}
}

func (r *nodeRepository) Create(ctx context.Context, node *domain.Node) error {
    if err := r.db.WithContext(ctx).Create(node).Error; err != nil {
        r.log.Errorw("node_repo_create_failed", "ip", node.IP, "error", err)
        return err
    }
    r.log.Infow("node_repo_create_ok", "id", node.ID, "ip", node.IP)
    return nil
}

func (r *nodeRepository) GetByID(ctx context.Context, id uint) (*domain.Node, error) {
    var node domain.Node
    if err := r.db.WithContext(ctx).First(&node, id).Error; err != nil {
        r.log.Errorw("node_repo_get_failed", "id", id, "error", err)
        return nil, err
    }
    return &node, nil
}

func (r *nodeRepository) GetByIP(ctx context.Context, ip string) (*domain.Node, error) {
    var node domain.Node
    if err := r.db.WithContext(ctx).Where("ip = ?", ip).First(&node).Error; err != nil {
        r.log.Errorw("node_repo_get_by_ip_failed", "ip", ip, "error", err)
        return nil, err
    }
    return &node, nil
}

func (r *nodeRepository) GetByIPWithDeleted(ctx context.Context, ip string) (*domain.Node, error) {
    var node domain.Node
    if err := r.db.WithContext(ctx).Unscoped().Where("ip = ?", ip).First(&node).Error; err != nil {
        r.log.Errorw("node_repo_get_by_ip_with_deleted_failed", "ip", ip, "error", err)
        return nil, err
    }
    return &node, nil
}

func (r *nodeRepository) GetAll(ctx context.Context) ([]domain.Node, error) {
    var nodes []domain.Node
    if err := r.db.WithContext(ctx).Find(&nodes).Error; err != nil {
        r.log.Errorw("node_repo_list_failed", "error", err)
        return nil, err
    }
    r.log.Infow("node_repo_list_ok", "count", len(nodes))
    return nodes, nil
}

func (r *nodeRepository) Update(ctx context.Context, node *domain.Node) error {
    if err := r.db.WithContext(ctx).Save(node).Error; err != nil {
        r.log.Errorw("node_repo_update_failed", "id", node.ID, "error", err)
        return err
    }
    r.log.Infow("node_repo_update_ok", "id", node.ID)
    return nil
}

func (r *nodeRepository) Restore(ctx context.Context, node *domain.Node) error {
    node.DeletedAt = gorm.DeletedAt{}
    if err := r.db.WithContext(ctx).Unscoped().Save(node).Error; err != nil {
        r.log.Errorw("node_repo_restore_failed", "id", node.ID, "error", err)
        return err
    }
    r.log.Infow("node_repo_restore_ok", "id", node.ID)
    return nil
}

func (r *nodeRepository) Delete(ctx context.Context, id uint) error {
    if err := r.db.WithContext(ctx).Delete(&domain.Node{}, id).Error; err != nil {
        r.log.Errorw("node_repo_delete_failed", "id", id, "error", err)
        return err
    }
    r.log.Infow("node_repo_delete_ok", "id", id)
    return nil
}
