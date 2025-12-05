package ports

import (
	"context"

	"github.com/netly/backend/internal/domain"
)

type NodeRepository interface {
	Create(ctx context.Context, node *domain.Node) error
	GetByID(ctx context.Context, id uint) (*domain.Node, error)
	GetByIP(ctx context.Context, ip string) (*domain.Node, error)
	GetByIPWithDeleted(ctx context.Context, ip string) (*domain.Node, error)
	GetAll(ctx context.Context) ([]domain.Node, error)
	Update(ctx context.Context, node *domain.Node) error
	Restore(ctx context.Context, node *domain.Node) error
	Delete(ctx context.Context, id uint) error
}

type TunnelRepository interface {
	Create(ctx context.Context, tunnel *domain.Tunnel) error
	GetByID(ctx context.Context, id uint) (*domain.Tunnel, error)
	GetByNodeID(ctx context.Context, nodeID uint) ([]domain.Tunnel, error)
	GetAll(ctx context.Context) ([]domain.Tunnel, error)
	Update(ctx context.Context, tunnel *domain.Tunnel) error
	Delete(ctx context.Context, id uint) error
}

type ServiceRepository interface {
	Create(ctx context.Context, service *domain.Service) error
	GetByID(ctx context.Context, id uint) (*domain.Service, error)
	GetByNodeID(ctx context.Context, nodeID uint) ([]domain.Service, error)
	GetAll(ctx context.Context) ([]domain.Service, error)
	Update(ctx context.Context, service *domain.Service) error
	Delete(ctx context.Context, id uint) error
}

type TimelineRepository interface {
	Create(ctx context.Context, event *domain.TimelineEvent) error
	GetByID(ctx context.Context, id uint) (*domain.TimelineEvent, error)
	GetByResource(ctx context.Context, resourceType string, resourceID uint) ([]domain.TimelineEvent, error)
	GetAll(ctx context.Context, limit int) ([]domain.TimelineEvent, error)
	Update(ctx context.Context, event *domain.TimelineEvent) error
}

type SystemSettingRepository interface {
	Get(ctx context.Context, key string) (*domain.SystemSetting, error)
	Set(ctx context.Context, setting *domain.SystemSetting) error
	GetByCategory(ctx context.Context, category string) ([]domain.SystemSetting, error)
	Delete(ctx context.Context, key string) error
}
