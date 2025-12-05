package db

import (
    "context"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "gorm.io/gorm"
)

type tunnelRepository struct {
    db  *gorm.DB
    log *logger.Logger
}

func NewTunnelRepository(db *gorm.DB, log *logger.Logger) ports.TunnelRepository {
    return &tunnelRepository{db: db, log: log}
}

func (r *tunnelRepository) Create(ctx context.Context, tunnel *domain.Tunnel) error {
    if err := r.db.WithContext(ctx).Create(tunnel).Error; err != nil {
        r.log.Errorw("tunnel_repo_create_failed", "source_node_id", tunnel.SourceNodeID, "dest_node_id", tunnel.DestNodeID, "error", err)
        return err
    }
    r.log.Infow("tunnel_repo_create_ok", "id", tunnel.ID)
    return nil
}

func (r *tunnelRepository) GetByID(ctx context.Context, id uint) (*domain.Tunnel, error) {
    var tunnel domain.Tunnel
    if err := r.db.WithContext(ctx).
        Preload("SourceNode").
        Preload("DestNode").
        First(&tunnel, id).Error; err != nil {
        r.log.Errorw("tunnel_repo_get_failed", "id", id, "error", err)
        return nil, err
    }
    return &tunnel, nil
}

func (r *tunnelRepository) GetByNodeID(ctx context.Context, nodeID uint) ([]domain.Tunnel, error) {
    var tunnels []domain.Tunnel
    if err := r.db.WithContext(ctx).
        Where("source_node_id = ? OR dest_node_id = ?", nodeID, nodeID).
        Find(&tunnels).Error; err != nil {
        r.log.Errorw("tunnel_repo_get_by_node_failed", "node_id", nodeID, "error", err)
        return nil, err
    }
    r.log.Infow("tunnel_repo_get_by_node_ok", "node_id", nodeID, "count", len(tunnels))
    return tunnels, nil
}

func (r *tunnelRepository) GetAll(ctx context.Context) ([]domain.Tunnel, error) {
    var tunnels []domain.Tunnel
    if err := r.db.WithContext(ctx).
        Preload("SourceNode").
        Preload("DestNode").
        Find(&tunnels).Error; err != nil {
        r.log.Errorw("tunnel_repo_list_failed", "error", err)
        return nil, err
    }
    r.log.Infow("tunnel_repo_list_ok", "count", len(tunnels))
    return tunnels, nil
}

func (r *tunnelRepository) Update(ctx context.Context, tunnel *domain.Tunnel) error {
    if err := r.db.WithContext(ctx).Save(tunnel).Error; err != nil {
        r.log.Errorw("tunnel_repo_update_failed", "id", tunnel.ID, "error", err)
        return err
    }
    r.log.Infow("tunnel_repo_update_ok", "id", tunnel.ID)
    return nil
}

func (r *tunnelRepository) Delete(ctx context.Context, id uint) error {
    if err := r.db.WithContext(ctx).Delete(&domain.Tunnel{}, id).Error; err != nil {
        r.log.Errorw("tunnel_repo_delete_failed", "id", id, "error", err)
        return err
    }
    r.log.Infow("tunnel_repo_delete_ok", "id", id)
    return nil
}
