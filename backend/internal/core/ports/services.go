package ports

import (
	"context"

	"github.com/netly/backend/internal/domain"
)

type NodeService interface {
	CreateNode(ctx context.Context, input CreateNodeInput) (*domain.Node, error)
	GetNodes(ctx context.Context) ([]domain.Node, error)
	GetNodeByID(ctx context.Context, id uint) (*domain.Node, error)
	DeleteNode(ctx context.Context, id uint) error
	UpdateNodeStatus(ctx context.Context, id uint, status domain.NodeStatus) error
	InstallAgent(ctx context.Context, id uint) error
	InstallAgentAsync(ctx context.Context, nodeID uint) (string, error) // Added Async method
	GetTaskStatus(taskID string) (*domain.Task, error)                  // Added Task status retrieval
	GetNodeAuth(ctx context.Context, id uint) (user, password, sshKey string, err error)
	UpdateNodeStats(ctx context.Context, id uint, stats domain.JSONB) error
}

type CreateNodeInput struct {
	Name     string
	IP       string
	SSHPort  int
	Role     domain.NodeRole
	User     string
	Password string
	SSHKey   string
	GeoData  domain.JSONB
}

type InstallerService interface {
	InstallAgent(ctx context.Context, node *domain.Node, authData string) error
	ValidateBinaryExistence() error
}

type TunnelService interface {
	CreateTunnel(ctx context.Context, input CreateTunnelInput) (*domain.Tunnel, error)
	CreateChain(ctx context.Context, entryID, relayID, exitID uint, protocol domain.TunnelProtocol) (*domain.Tunnel, error)
	GetTunnels(ctx context.Context) ([]domain.Tunnel, error)
	GetTunnelByID(ctx context.Context, id uint) (*domain.Tunnel, error)
	DeleteTunnel(ctx context.Context, id uint) error
}

type CreateTunnelInput struct {
	Name         string
	Protocol     domain.TunnelProtocol
	SourceNodeID uint
	DestNodeID   uint
	SourcePort   int
	DestPort     int
}

type IPAMService interface {
	AllocateTunnelIPs(ctx context.Context) (ipv4 string, ipv6 string, err error)
	ReleaseIPs(ctx context.Context, ipv4, ipv6 string) error
}

type PortAMService interface {
	ReservePort(ctx context.Context, nodeID uint, protocol string) (int, error)
	ReleasePort(ctx context.Context, nodeID uint, port int, protocol string) error
	IsPortAvailable(ctx context.Context, nodeID uint, port int, protocol string) (bool, error)
}

type ServiceService interface {
	CreateService(ctx context.Context, input CreateServiceInput) (*domain.Service, error)
	GetServices(ctx context.Context) ([]domain.Service, error)
	GetServiceByID(ctx context.Context, id uint) (*domain.Service, error)
	DeleteService(ctx context.Context, id uint) error
}

type CreateServiceInput struct {
	Name        string
	Protocol    domain.ServiceProtocol
	NodeID      uint
	ListenPort  int
	RoutingMode domain.RoutingMode
	Config      domain.JSONB
}
